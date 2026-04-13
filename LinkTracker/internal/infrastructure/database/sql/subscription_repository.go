package sqlrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

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

	const deleteLinkChatQuery = `
		DELETE FROM link_chat
		WHERE chat_id = $1 AND link_id = $2
	`

	tag, err := tx.Exec(ctx, deleteLinkChatQuery, chatID, linkID)
	if err != nil {
		return fmt.Errorf("delete link_chat relation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrLinkNotFound
	}

	// If there is no chat tracking this link, remove link from links
	const countRelationsQuery = `
		SELECT COUNT(*)
		FROM link_chat
		WHERE link_id = $1
	`

	var relationsCount int
	if err := tx.QueryRow(ctx, countRelationsQuery, linkID).Scan(&relationsCount); err != nil {
		return fmt.Errorf("count link relations: %w", err)
	}

	if relationsCount == 0 {
		const deleteLinkQuery = `
			DELETE FROM links
			WHERE id = $1
		`

		if _, err := tx.Exec(ctx, deleteLinkQuery, linkID); err != nil {
			return fmt.Errorf("delete link: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit remove link transaction: %w", err)
	}

	return nil
}

func (r *SubscriptionRepository) GetLink(ctx context.Context, chatID int64, url string) (domain.RepositoryLink, error) {
	const query = `
		SELECT l.id, l.url
		FROM links l
		INNER JOIN link_chat lc ON lc.link_id = l.id
		WHERE lc.chat_id = $1 AND l.url = $2
	`

	var link domain.RepositoryLink
	err := r.pool.QueryRow(ctx, query, chatID, url).Scan(&link.ID, &link.URL)
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
	const query = `
		SELECT l.id, l.url, lt.tag
		FROM (
			SELECT l.id, l.url
			FROM links l
			INNER JOIN link_chat lc ON lc.link_id = l.id
			WHERE lc.chat_id = $1
			ORDER BY l.id
			LIMIT $2 OFFSET $3
		) l
		LEFT JOIN link_tag lt
			ON lt.link_id = l.id AND lt.chat_id = $1
		ORDER BY l.id, lt.tag
	`

	rows, err := r.pool.Query(ctx, query, chatID, limit, offset)
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
	const query = `
		SELECT lc.chat_id
		FROM link_chat lc
		INNER JOIN links l ON l.id = lc.link_id
		WHERE l.url = $1
		ORDER BY lc.chat_id
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, url, limit, offset)
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
	const query = `
		SELECT url, last_updated_at
		FROM links
		ORDER BY id
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tracked urls: %w", err)
	}
	defer rows.Close()

	tracked := make(map[string]time.Time)
	for rows.Next() {
		var url string
		var updatedAt *time.Time
		if err := rows.Scan(&url, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan tracked url: %w", err)
		}
		tracked[url] = *updatedAt
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracked urls: %w", err)
	}

	return tracked, nil
}

func (r *SubscriptionRepository) UpdateLastUpdated(ctx context.Context, url string, updatedAt time.Time) error {
	const query = `
		UPDATE links
		SET last_updated_at = $2
		WHERE url = $1
	`

	tag, err := r.pool.Exec(ctx, query, url, updatedAt)
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

	const query = `
		INSERT INTO link_tag (chat_id, link_id, tag)
		VALUES ($1, $2, $3)
	`

	_, err = r.pool.Exec(ctx, query, chatID, linkID, tag)
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

	const query = `
		DELETE FROM link_tag
		WHERE chat_id = $1 AND link_id = $2 AND tag = $3
	`

	cmdTag, err := r.pool.Exec(ctx, query, chatID, linkID, tag)
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

	const query = `
		SELECT tag
		FROM link_tag
		WHERE chat_id = $1 AND link_id = $2
		ORDER BY tag
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, query, chatID, linkID, limit, offset)
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
	const insertLinkQuery = `
		INSERT INTO links (url)
		VALUES ($1)
		ON CONFLICT (url) DO NOTHING
	`

	if _, err := tx.Exec(ctx, insertLinkQuery, url); err != nil {
		return 0, fmt.Errorf("insert link: %w", err)
	}

	return r.getLinkID(ctx, tx, url)
}

func (r *SubscriptionRepository) getLinkID(ctx context.Context, tx pgx.Tx, url string) (int64, error) {
	const query = `
		SELECT id
		FROM links
		WHERE url = $1
	`

	var linkID int64
	err := tx.QueryRow(ctx, query, url).Scan(&linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, repository.ErrLinkNotFound
		}
		return 0, fmt.Errorf("get link id: %w", err)
	}

	return linkID, nil
}

func (r *SubscriptionRepository) getLinkIDForUpdate(ctx context.Context, tx pgx.Tx, url string) (int64, error) {
	const query = `
		SELECT id
		FROM links
		WHERE url = $1
		FOR UPDATE
	`

	var linkID int64
	err := tx.QueryRow(ctx, query, url).Scan(&linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, repository.ErrLinkNotFound
		}

		return 0, fmt.Errorf("get link id for update: %w", err)
	}

	return linkID, nil
}

func (r *SubscriptionRepository) getLinkIDByChatAndURL(ctx context.Context, chatID int64, url string) (int64, error) {
	const query = `
		SELECT l.id
		FROM links l
		INNER JOIN link_chat lc ON lc.link_id = l.id
		WHERE lc.chat_id = $1 AND l.url = $2
	`

	var linkID int64
	err := r.pool.QueryRow(ctx, query, chatID, url).Scan(&linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, repository.ErrLinkNotFound
		}
		return 0, fmt.Errorf("get link id by chat and url: %w", err)
	}

	return linkID, nil
}

func (r *SubscriptionRepository) insertLinkChat(ctx context.Context, tx pgx.Tx, chatID int64, linkID int64) error {
	const query = `
		INSERT INTO link_chat (chat_id, link_id)
		VALUES ($1, $2)
	`

	_, err := tx.Exec(ctx, query, chatID, linkID)
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

	const query = `
		INSERT INTO link_tag (chat_id, link_id, tag)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id, link_id, tag) DO NOTHING
	`

	for _, tag := range tags {
		if _, err := tx.Exec(ctx, query, chatID, linkID, tag); err != nil {
			return fmt.Errorf("insert tag relation: %w", err)
		}
	}

	return nil
}

func (r *SubscriptionRepository) getTagsByChatAndLinkID(ctx context.Context, chatID int64, linkID int64) ([]string, error) {
	const query = `
		SELECT tag
		FROM link_tag
		WHERE chat_id = $1 AND link_id = $2
		ORDER BY tag
	`

	rows, err := r.pool.Query(ctx, query, chatID, linkID)
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
