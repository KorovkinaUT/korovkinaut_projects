package goqurepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
)

var dialect = goqu.Dialect("postgres")

// Implementation of ChatRepository Interface
// Stores registered chats
type ChatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{
		pool: pool,
	}
}

func (r *ChatRepository) Register(ctx context.Context, chatID int64) error {
	ds := dialect.
		Insert("chats").
		Prepared(true).
		Rows(goqu.Record{
			"id": chatID,
		})

	sql, args, err := ds.ToSQL()
	if err != nil {
		return fmt.Errorf("build register chat query: %w", err)
	}

	_, err = r.pool.Exec(ctx, sql, args...)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return repository.ErrChatAlreadyExists
	}

	return fmt.Errorf("register chat: %w", err)
}

func (r *ChatRepository) Delete(ctx context.Context, chatID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete chat transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	deleteChatDS := dialect.
		Delete("chats").
		Prepared(true).
		Where(goqu.C("id").Eq(chatID))

	sql, args, err := deleteChatDS.ToSQL()
	if err != nil {
		return fmt.Errorf("build delete chat query: %w", err)
	}

	tag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrChatNotFound
	}

	// ON DELETE CASCADE deletes rows from link_chat and link_tag but not from links
	subQuery := dialect.
		From("link_chat").
		Prepared(true).
		Select(goqu.C("link_id"))

	deleteOrphanLinksDS := dialect.
		Delete("links").
		Prepared(true).
		Where(goqu.C("id").NotIn(subQuery))

	sql, args, err = deleteOrphanLinksDS.ToSQL()
	if err != nil {
		return fmt.Errorf("build delete orphan links query: %w", err)
	}

	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("delete orphan links: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete chat transaction: %w", err)
	}

	return nil
}

func (r *ChatRepository) Exists(ctx context.Context, chatID int64) (bool, error) {
	ds := dialect.
		From("chats").
		Prepared(true).
		Select(goqu.L("1")).
		Where(goqu.C("id").Eq(chatID)).
		Limit(1)

	sql, args, err := ds.ToSQL()
	if err != nil {
		return false, fmt.Errorf("build check chat exists query: %w", err)
	}

	var exists int
	err = r.pool.QueryRow(ctx, sql, args...).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check chat exists: %w", err)
	}

	return true, nil
}

var _ repository.ChatRepository = (*ChatRepository)(nil)
