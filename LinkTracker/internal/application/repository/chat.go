package repository

import "errors"

var (
	ErrChatAlreadyExists = errors.New("chat already exists")
	ErrChatNotFound      = errors.New("chat not found")
)

type ChatRepository interface {
	Register(chatID int64) error
	Delete(chatID int64) error
	Exists(chatID int64) bool
}
