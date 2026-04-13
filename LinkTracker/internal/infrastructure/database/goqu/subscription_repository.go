package goqurepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

// Implementation of SubscriptionRepository Interface
// Stores chats and links matching
type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{
		pool: pool,
	}
}

func (r *SubscriptionRepository) AddLink(ctx context.Context, chatID int64, url string, tags []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add link transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	linkID, err := r.insertLink(ctx, tx, url)
	if err != nil {
		return err
	}

	if err := r.insertLinkChat(ctx, tx, chatID, linkID); err != nil {
		return err
	}

	if err := r.insertTags(ctx, tx, chatID, linkID, tags); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit add link transaction: %w", err)
	}

	return nil
}

func (r *SubscriptionRepository) RemoveLink(ctx context.Context, chatID int64, url string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin remove link transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	linkID, err := r.getLinkIDForUpdate(ctx, tx, url)
	if err != nil {
		return err
	}

	deleteLinkChatDS := dialect.
		Delete("link_chat").
		Prepared(true).
		Where(
			goqu.C("chat_id").Eq(chatID),
			goqu.C("link_id").Eq(linkID),
		)

	sql, args, err := deleteLinkChatDS.ToSQL()
	if err != nil {
		return fmt.Errorf("build delete link_chat relation query: %w", err)
	}

	tag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("delete link_chat relation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrLinkNotFound
	}

	// If there is no chat tracking this link, remove link from links
	countRelationsDS := dialect.
		From("link_chat").
		Prepared(true).
		Select(goqu.COUNT("*")).
		Where(goqu.C("link_id").Eq(linkID))

	sql, args, err = countRelationsDS.ToSQL()
	if err != nil {
		return fmt.Errorf("build count link relations query: %w", err)
	}

	var relationsCount int
	if err := tx.QueryRow(ctx, sql, args...).Scan(&relationsCount); err != nil {
		return fmt.Errorf("count link relations: %w", err)
	}

	if relationsCount == 0 {
		deleteLinkDS := dialect.
			Delete("links").
			Prepared(true).
			Where(goqu.C("id").Eq(linkID))

		sql, args, err = deleteLinkDS.ToSQL()
		if err != nil {
			return fmt.Errorf("build delete link query: %w", err)
		}

		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return fmt.Errorf("delete link: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit remove link transaction: %w", err)
	}

	return nil
}

func (r *SubscriptionRepository) GetLink(ctx context.Context, chatID int64, url string) (domain.RepositoryLink, error) {
	queryDS := dialect.
		From(goqu.T("links").As("l")).
		Prepared(true).
		InnerJoin(
			goqu.T("link_chat").As("lc"),
			goqu.On(goqu.I("lc.link_id").Eq(goqu.I("l.id"))),
		).
		Select(goqu.I("l.id"), goqu.I("l.url")).
		Where(
			goqu.I("lc.chat_id").Eq(chatID),
			goqu.I("l.url").Eq(url),
		)

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return domain.RepositoryLink{}, fmt.Errorf("build get link query: %w", err)
	}

	var link domain.RepositoryLink
	err = r.pool.QueryRow(ctx, sql, args...).Scan(&link.ID, &link.URL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RepositoryLink{}, repository.ErrLinkNotFound
		}

		return domain.RepositoryLink{}, fmt.Errorf("get link: %w", err)
	}

	tags, err := r.getTagsByChatAndLinkID(ctx, chatID, link.ID)
	if err != nil {
		return domain.RepositoryLink{}, err
	}

	link.Tags = tags

	return link, nil
}

func (r *SubscriptionRepository) ListLinks(ctx context.Context, chatID int64, limit int64, offset int64) ([]domain.RepositoryLink, error) {
	linksDS := dialect.
		From(goqu.T("links").As("l")).
		Prepared(true).
		InnerJoin(
			goqu.T("link_chat").As("lc"),
			goqu.On(goqu.I("lc.link_id").Eq(goqu.I("l.id"))),
		).
		Select(
			goqu.I("l.id").As("id"),
			goqu.I("l.url").As("url"),
		).
		Where(goqu.I("lc.chat_id").Eq(chatID)).
		Order(goqu.I("l.id").Asc()).
		Limit(uint(limit)).
		Offset(uint(offset)).
		As("links_page")

	queryDS := dialect.
		From(linksDS).
		Prepared(true).
		LeftJoin(
			goqu.T("link_tag").As("lt"),
			goqu.On(
				goqu.I("lt.link_id").Eq(goqu.I("links_page.id")),
				goqu.I("lt.chat_id").Eq(chatID),
			),
		).
		Select(
			goqu.I("links_page.id"),
			goqu.I("links_page.url"),
			goqu.I("lt.tag"),
		).
		Order(
			goqu.I("links_page.id").Asc(),
			goqu.I("lt.tag").Asc(),
		)

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list links query: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	defer rows.Close()

	links := make([]domain.RepositoryLink, 0)
	linkIndex := make(map[int64]int)

	for rows.Next() {
		var (
			linkID int64
			url    string
			tag    *string
		)

		if err := rows.Scan(&linkID, &url, &tag); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}

		idx, exists := linkIndex[linkID]
		if !exists {
			links = append(links, domain.RepositoryLink{
				ID:   linkID,
				URL:  url,
				Tags: make([]string, 0),
			})
			idx = len(links) - 1
			linkIndex[linkID] = idx
		}

		if tag != nil {
			links[idx].Tags = append(links[idx].Tags, *tag)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate links: %w", err)
	}

	return links, nil
}

func (r *SubscriptionRepository) ListChatIDs(ctx context.Context, url string, limit int64, offset int64) ([]int64, error) {
	queryDS := dialect.
		From(goqu.T("link_chat").As("lc")).
		Prepared(true).
		InnerJoin(
			goqu.T("links").As("l"),
			goqu.On(goqu.I("l.id").Eq(goqu.I("lc.link_id"))),
		).
		Select(goqu.I("lc.chat_id")).
		Where(goqu.I("l.url").Eq(url)).
		Order(goqu.I("lc.chat_id").Asc()).
		Limit(uint(limit)).
		Offset(uint(offset))

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list chat ids by url query: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list chat ids by url: %w", err)
	}
	defer rows.Close()

	chatIDs := make([]int64, 0)
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, fmt.Errorf("scan chat id: %w", err)
		}
		chatIDs = append(chatIDs, chatID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat ids: %w", err)
	}

	return chatIDs, nil
}

func (r *SubscriptionRepository) ListTrackedURLs(ctx context.Context, limit int64, offset int64) (map[string]time.Time, error) {
	queryDS := dialect.
		From("links").
		Prepared(true).
		Select("url", "last_updated_at").
		Order(goqu.C("id").Asc()).
		Limit(uint(limit)).
		Offset(uint(offset))

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list tracked urls query: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list tracked urls: %w", err)
	}
	defer rows.Close()

	tracked := make(map[string]time.Time)
	for rows.Next() {
		var url string
		var updatedAt time.Time
		if err := rows.Scan(&url, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan tracked url: %w", err)
		}
		tracked[url] = updatedAt
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracked urls: %w", err)
	}

	return tracked, nil
}

func (r *SubscriptionRepository) UpdateLastUpdated(ctx context.Context, url string, updatedAt time.Time) error {
	queryDS := dialect.
		Update("links").
		Prepared(true).
		Set(goqu.Record{
			"last_updated_at": updatedAt,
		}).
		Where(goqu.C("url").Eq(url))

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return fmt.Errorf("build update last updated query: %w", err)
	}

	tag, err := r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("update last updated: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrLinkNotFound
	}

	return nil
}

func (r *SubscriptionRepository) AddTag(ctx context.Context, chatID int64, url string, tag string) error {
	linkID, err := r.getLinkIDByChatAndURL(ctx, chatID, url)
	if err != nil {
		return err
	}

	queryDS := dialect.
		Insert("link_tag").
		Prepared(true).
		Rows(goqu.Record{
			"chat_id": chatID,
			"link_id": linkID,
			"tag":     tag,
		})

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return fmt.Errorf("build add tag query: %w", err)
	}

	_, err = r.pool.Exec(ctx, sql, args...)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return repository.ErrTagAlreadyExists
		case "23503":
			return repository.ErrLinkNotFound
		}
	}

	return fmt.Errorf("add tag: %w", err)
}

func (r *SubscriptionRepository) RemoveTag(ctx context.Context, chatID int64, url string, tag string) error {
	linkID, err := r.getLinkIDByChatAndURL(ctx, chatID, url)
	if err != nil {
		return err
	}

	queryDS := dialect.
		Delete("link_tag").
		Prepared(true).
		Where(
			goqu.C("chat_id").Eq(chatID),
			goqu.C("link_id").Eq(linkID),
			goqu.C("tag").Eq(tag),
		)

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return fmt.Errorf("build remove tag query: %w", err)
	}

	cmdTag, err := r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("remove tag: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return repository.ErrTagNotFound
	}

	return nil
}

func (r *SubscriptionRepository) ListTags(ctx context.Context, chatID int64, url string, limit int64, offset int64) ([]string, error) {
	linkID, err := r.getLinkIDByChatAndURL(ctx, chatID, url)
	if err != nil {
		return nil, err
	}

	queryDS := dialect.
		From("link_tag").
		Prepared(true).
		Select("tag").
		Where(
			goqu.C("chat_id").Eq(chatID),
			goqu.C("link_id").Eq(linkID),
		).
		Order(goqu.C("tag").Asc()).
		Limit(uint(limit)).
		Offset(uint(offset))

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list tags query: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	tags := make([]string, 0)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	return tags, nil
}

func (r *SubscriptionRepository) insertLink(ctx context.Context, tx pgx.Tx, url string) (int64, error) {
	insertLinkDS := dialect.
		Insert("links").
		Prepared(true).
		Rows(goqu.Record{
			"url": url,
		}).
		OnConflict(goqu.DoNothing())

	sql, args, err := insertLinkDS.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("build insert link query: %w", err)
	}

	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return 0, fmt.Errorf("insert link: %w", err)
	}

	return r.getLinkID(ctx, tx, url)
}

func (r *SubscriptionRepository) getLinkID(ctx context.Context, tx pgx.Tx, url string) (int64, error) {
	queryDS := dialect.
		From("links").
		Prepared(true).
		Select("id").
		Where(goqu.C("url").Eq(url))

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("build get link id query: %w", err)
	}

	var linkID int64
	err = tx.QueryRow(ctx, sql, args...).Scan(&linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, repository.ErrLinkNotFound
		}
		return 0, fmt.Errorf("get link id: %w", err)
	}

	return linkID, nil
}

func (r *SubscriptionRepository) getLinkIDForUpdate(ctx context.Context, tx pgx.Tx, url string) (int64, error) {
	queryDS := dialect.
		From("links").
		Prepared(true).
		Select("id").
		Where(goqu.C("url").Eq(url)).
		ForUpdate(goqu.Wait)

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("build get link id for update query: %w", err)
	}

	var linkID int64
	err = tx.QueryRow(ctx, sql, args...).Scan(&linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, repository.ErrLinkNotFound
		}

		return 0, fmt.Errorf("get link id for update: %w", err)
	}

	return linkID, nil
}

func (r *SubscriptionRepository) getLinkIDByChatAndURL(ctx context.Context, chatID int64, url string) (int64, error) {
	queryDS := dialect.
		From(goqu.T("links").As("l")).
		Prepared(true).
		InnerJoin(
			goqu.T("link_chat").As("lc"),
			goqu.On(goqu.I("lc.link_id").Eq(goqu.I("l.id"))),
		).
		Select(goqu.I("l.id")).
		Where(
			goqu.I("lc.chat_id").Eq(chatID),
			goqu.I("l.url").Eq(url),
		)

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("build get link id by chat and url query: %w", err)
	}

	var linkID int64
	err = r.pool.QueryRow(ctx, sql, args...).Scan(&linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, repository.ErrLinkNotFound
		}
		return 0, fmt.Errorf("get link id by chat and url: %w", err)
	}

	return linkID, nil
}

func (r *SubscriptionRepository) insertLinkChat(ctx context.Context, tx pgx.Tx, chatID int64, linkID int64) error {
	queryDS := dialect.
		Insert("link_chat").
		Prepared(true).
		Rows(goqu.Record{
			"chat_id": chatID,
			"link_id": linkID,
		})

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return fmt.Errorf("build insert link_chat relation query: %w", err)
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return repository.ErrLinkAlreadyTracked
		case "23503":
			switch pgErr.ConstraintName {
			case "link_chat_chat_id_fkey":
				return repository.ErrChatNotFound
			case "link_chat_link_id_fkey":
				return repository.ErrLinkNotFound
			}
		}
	}

	return fmt.Errorf("insert link_chat relation: %w", err)
}

func (r *SubscriptionRepository) insertTags(ctx context.Context, tx pgx.Tx, chatID int64, linkID int64, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	queryDS := dialect.
		Insert("link_tag").
		Prepared(true).
		OnConflict(goqu.DoNothing())

	for _, tag := range tags {
		sql, args, err := queryDS.
			Rows(goqu.Record{
				"chat_id": chatID,
				"link_id": linkID,
				"tag":     tag,
			}).
			ToSQL()
		if err != nil {
			return fmt.Errorf("build insert tag relation query: %w", err)
		}

		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return fmt.Errorf("insert tag relation: %w", err)
		}
	}

	return nil
}

func (r *SubscriptionRepository) getTagsByChatAndLinkID(ctx context.Context, chatID int64, linkID int64) ([]string, error) {
	queryDS := dialect.
		From("link_tag").
		Prepared(true).
		Select("tag").
		Where(
			goqu.C("chat_id").Eq(chatID),
			goqu.C("link_id").Eq(linkID),
		).
		Order(goqu.C("tag").Asc())

	sql, args, err := queryDS.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build get link tags query: %w", err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("get link tags: %w", err)
	}
	defer rows.Close()

	tags := make([]string, 0)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan link tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate link tags: %w", err)
	}

	return tags, nil
}

var _ repository.SubscriptionRepository = (*SubscriptionRepository)(nil)
