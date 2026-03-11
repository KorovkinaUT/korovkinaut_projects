package dispatcher

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
)

func TestDispatcher_ListCommand_WithSubscriptions(t *testing.T) {
	listHandler := NewList(func(chatID int64) (scrapperhttp.ListLinksResponse, error) {
		if chatID != 1 {
			t.Errorf("unexpected chatID: got %d, want 1", chatID)
		}

		return scrapperhttp.ListLinksResponse{
			Links: []scrapperhttp.LinkResponse{
				{
					ID:   1,
					URL:  "https://github.com/user/repo",
					Tags: []string{"backend", "go"},
				},
				{
					ID:   2,
					URL:  "https://stackoverflow.com/questions/123/test",
					Tags: []string{"qa"},
				},
			},
			Size: 2,
		}, nil
	})

	dispatcher := NewDispatcher([]Handler{listHandler})
	msg := newCommandMessage(1, "/list")

	got := dispatcher.Dispatch(msg)

	want := "Отслеживаемые ссылки:\n" +
		"1. https://github.com/user/repo [теги: backend, go]\n" +
		"2. https://stackoverflow.com/questions/123/test [теги: qa]"

	if got != want {
		t.Errorf("unexpected response:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestDispatcher_ListCommand_EmptyList(t *testing.T) {
	listHandler := NewList(func(chatID int64) (scrapperhttp.ListLinksResponse, error) {
		return scrapperhttp.ListLinksResponse{}, repository.ErrChatNotFound
	})

	dispatcher := NewDispatcher([]Handler{listHandler})
	msg := newCommandMessage(1, "/list")

	got := dispatcher.Dispatch(msg)

	want := "Список отслеживаемых ссылок пуст."

	if got != want {
		t.Errorf("unexpected response:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestDispatcher_ListCommand_FilterByTag(t *testing.T) {
	listHandler := NewList(func(chatID int64) (scrapperhttp.ListLinksResponse, error) {
		return scrapperhttp.ListLinksResponse{
			Links: []scrapperhttp.LinkResponse{
				{
					ID:   1,
					URL:  "https://github.com/user/repo",
					Tags: []string{"backend", "go"},
				},
				{
					ID:   2,
					URL:  "https://stackoverflow.com/questions/123/test",
					Tags: []string{"qa"},
				},
			},
			Size: 2,
		}, nil
	})

	dispatcher := NewDispatcher([]Handler{listHandler})
	msg := newCommandMessage(1, "/list qa")

	got := dispatcher.Dispatch(msg)

	want := "Отслеживаемые ссылки:\n" +
		"1. https://stackoverflow.com/questions/123/test [теги: qa]"

	if got != want {
		t.Errorf("unexpected response:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestDispatcher_ListCommand_InternalError(t *testing.T) {
	listHandler := NewList(func(chatID int64) (scrapperhttp.ListLinksResponse, error) {
		return scrapperhttp.ListLinksResponse{}, errors.New("internal error")
	})

	dispatcher := NewDispatcher([]Handler{listHandler})
	msg := newCommandMessage(1, "/list")

	got := dispatcher.Dispatch(msg)

	want := "Не удалось получить список ссылок."

	if got != want {
		t.Errorf("unexpected response:\nwant: %q\ngot:  %q", want, got)
	}
}

func newCommandMessage(chatID int64, text string) *tgbotapi.Message {
	return &tgbotapi.Message{
		Text: text,
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
		Entities: []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Offset: 0,
				Length: commandLength(text),
			},
		},
	}
}

func commandLength(text string) int {
	for i, r := range text {
		if r == ' ' {
			return i
		}
	}
	return len(text)
}