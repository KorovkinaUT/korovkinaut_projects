package memory

import (
	"sync"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
)

// Implementation of ChatRepository Interface
// Stores registered chats
type ChatRepository struct {
	mu    sync.RWMutex
	chats map[int64]struct{}
}

func NewChatRepository() *ChatRepository {
	return &ChatRepository{
		chats: make(map[int64]struct{}),
	}
}

func (r *ChatRepository) Register(chatID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.chats[chatID]; exists {
		return repository.ErrChatAlreadyExists
	}

	r.chats[chatID] = struct{}{}
	return nil
}

func (r *ChatRepository) Delete(chatID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.chats[chatID]; !exists {
		return repository.ErrChatNotFound
	}

	delete(r.chats, chatID)
	return nil
}

func (r *ChatRepository) Exists(chatID int64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.chats[chatID]
	return exists
}

// CompileTime check of methods correctness
var _ repository.ChatRepository = (*ChatRepository)(nil)