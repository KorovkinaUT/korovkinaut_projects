package repository

import (
	"context"
	"errors"
)

var (
	ErrChatAlreadyExists = errors.New("chat already exists")
	ErrChatNotFound      = errors.New("chat not found")
)

type ChatRepository interface {
	Register(ctx context.Context, chatID int64) error
	Delete(ctx context.Context, chatID int64) error
	Exists(ctx context.Context, chatID int64) (bool, error)
}
