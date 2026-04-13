package sqlrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
)

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
	const query = `
		INSERT INTO chats (id)
		VALUES ($1)
	`

	_, err := r.pool.Exec(ctx, query, chatID)
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

	const deleteChatQuery = `
		DELETE FROM chats
		WHERE id = $1
	`

	tag, err := tx.Exec(ctx, deleteChatQuery, chatID)
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrChatNotFound
	}

	// ON DELETE CASCADE deletes rows from link_chat and link_tag but not from links
	const deleteOrphanLinksQuery = `
		DELETE FROM links l
		WHERE NOT EXISTS (
			SELECT 1
			FROM link_chat lc
			WHERE lc.link_id = l.id
		)
	`

	if _, err := tx.Exec(ctx, deleteOrphanLinksQuery); err != nil {
		return fmt.Errorf("delete orphan links: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete chat transaction: %w", err)
	}

	return nil
}

func (r *ChatRepository) Exists(ctx context.Context, chatID int64) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM chats
			WHERE id = $1
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, chatID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check chat exists: %w", err)
	}

	return exists, nil
}

var _ repository.ChatRepository = (*ChatRepository)(nil)
