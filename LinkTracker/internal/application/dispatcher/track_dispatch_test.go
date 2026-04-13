package dispatcher

import (
	"context"
	"reflect"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
)

func TestDispatcher_TrackCommand_Success(t *testing.T) {
	// arrange
	ctx := context.Background()
	parser := schedulerlink.NewService()

	var (
		addCalled bool
		gotChatID int64
		gotReq    scrapperhttp.AddLinkRequest
	)

	trackHandler := NewTrack(parser, func(ctx context.Context, chatID int64, request scrapperhttp.AddLinkRequest) (scrapperhttp.LinkResponse, error) {
		addCalled = true
		gotChatID = chatID
		gotReq = request

		return scrapperhttp.LinkResponse{
			ID:   1,
			URL:  request.Link,
			Tags: request.Tags,
		}, nil
	})

	dispatcher := NewDispatcher([]Handler{trackHandler})
	chatID := int64(1)

	// act
	gotStart := dispatcher.Dispatch(ctx, newCommandMessage(chatID, "/track"))
	gotLink := dispatcher.Dispatch(ctx, newTextMessage(chatID, "https://github.com/user/repo"))
	dialogAfterLink := trackHandler.States().Get(chatID)
	gotFinish := dispatcher.Dispatch(ctx, newTextMessage(chatID, "backend, go"))
	dialogAfterFinish := trackHandler.States().Get(chatID)

	// assert
	wantStart := "Отправьте ссылку для отслеживания."
	if gotStart != wantStart {
		t.Errorf("unexpected response on /track:\nwant: %q\ngot:  %q", wantStart, gotStart)
	}

	wantLink := "Введите теги через запятую, введите /cancel, чтобы не добавлять теги."
	if gotLink != wantLink {
		t.Errorf("unexpected response after valid link:\nwant: %q\ngot:  %q", wantLink, gotLink)
	}

	if dialogAfterLink.State != StateWaitingTrackTags {
		t.Errorf("unexpected state after valid link: got %v, want %v", dialogAfterLink.State, StateWaitingTrackTags)
	}

	if dialogAfterLink.Link != "https://github.com/user/repo" {
		t.Errorf("unexpected saved link: got %q, want %q", dialogAfterLink.Link, "https://github.com/user/repo")
	}

	wantFinish := "Ссылка добавлена в отслеживание."
	if gotFinish != wantFinish {
		t.Errorf("unexpected response after tags:\nwant: %q\ngot:  %q", wantFinish, gotFinish)
	}

	if !addCalled {
		t.Errorf("expected addLink to be called")
	}

	if gotChatID != chatID {
		t.Errorf("unexpected chatID in addLink: got %d, want %d", gotChatID, chatID)
	}

	wantReq := scrapperhttp.AddLinkRequest{
		Link: "https://github.com/user/repo",
		Tags: []string{"backend", "go"},
	}
	if !reflect.DeepEqual(gotReq, wantReq) {
		t.Errorf("unexpected addLink request:\nwant: %+v\ngot:  %+v", wantReq, gotReq)
	}

	if dialogAfterFinish.State != StateIdle {
		t.Errorf("unexpected state after finish: got %v, want %v", dialogAfterFinish.State, StateIdle)
	}
}

func TestDispatcher_TrackCommand_InvalidLink(t *testing.T) {
	// arrange
	ctx := context.Background()
	parser := schedulerlink.NewService()

	addCalled := false
	trackHandler := NewTrack(parser, func(ctx context.Context, chatID int64, request scrapperhttp.AddLinkRequest) (scrapperhttp.LinkResponse, error) {
		addCalled = true
		return scrapperhttp.LinkResponse{}, nil
	})

	dispatcher := NewDispatcher([]Handler{trackHandler})
	chatID := int64(1)

	startMsg := newCommandMessage(chatID, "/track")
	invalidLinkMsg := newTextMessage(chatID, "tbank://github.com/user/repo")

	// act
	gotStart := dispatcher.Dispatch(ctx, startMsg)
	gotInvalid := dispatcher.Dispatch(ctx, invalidLinkMsg)
	dialogAfterInvalid := trackHandler.States().Get(chatID)

	// assert
	wantStart := "Отправьте ссылку для отслеживания."
	if gotStart != wantStart {
		t.Errorf("unexpected response on /track:\nwant: %q\ngot:  %q", wantStart, gotStart)
	}

	wantInvalid := "Некорректная ссылка."
	if gotInvalid != wantInvalid {
		t.Errorf("unexpected response for invalid link:\nwant: %q\ngot:  %q", wantInvalid, gotInvalid)
	}

	if addCalled {
		t.Errorf("addLink must not be called for invalid link")
	}

	if dialogAfterInvalid.State != StateWaitingTrackLink {
		t.Errorf("unexpected state after invalid link: got %v, want %v", dialogAfterInvalid.State, StateWaitingTrackLink)
	}
}

func TestDispatcher_TrackCommand_AlreadyTracked(t *testing.T) {
	// arrange
	ctx := context.Background()
	parser := schedulerlink.NewService()

	trackHandler := NewTrack(parser, func(ctx context.Context, chatID int64, request scrapperhttp.AddLinkRequest) (scrapperhttp.LinkResponse, error) {
		return scrapperhttp.LinkResponse{}, repository.ErrLinkAlreadyTracked
	})

	dispatcher := NewDispatcher([]Handler{trackHandler})
	chatID := int64(1)

	startMsg := newCommandMessage(chatID, "/track")
	linkMsg := newTextMessage(chatID, "https://github.com/user/repo")
	tagsMsg := newTextMessage(chatID, "backend, go")

	// act
	gotStart := dispatcher.Dispatch(ctx, startMsg)
	gotLink := dispatcher.Dispatch(ctx, linkMsg)
	gotFinish := dispatcher.Dispatch(ctx, tagsMsg)

	// assert
	wantStart := "Отправьте ссылку для отслеживания."
	if gotStart != wantStart {
		t.Errorf("unexpected response on /track:\nwant: %q\ngot:  %q", wantStart, gotStart)
	}

	wantLink := "Введите теги через запятую, введите /cancel, чтобы не добавлять теги."
	if gotLink != wantLink {
		t.Errorf("unexpected response after valid link:\nwant: %q\ngot:  %q", wantLink, gotLink)
	}

	wantFinish := "Ссылка уже отслеживается"
	if gotFinish != wantFinish {
		t.Errorf("unexpected response for already tracked link:\nwant: %q\ngot:  %q", wantFinish, gotFinish)
	}
}

func TestDispatcher_TrackCommand_CancelMeansSaveWithoutTags(t *testing.T) {
	// arrange
	ctx := context.Background()
	parser := schedulerlink.NewService()

	var (
		addCalled bool
		gotChatID int64
		gotReq    scrapperhttp.AddLinkRequest
	)

	trackHandler := NewTrack(parser, func(ctx context.Context, chatID int64, request scrapperhttp.AddLinkRequest) (scrapperhttp.LinkResponse, error) {
		addCalled = true
		gotChatID = chatID
		gotReq = request

		return scrapperhttp.LinkResponse{
			ID:   1,
			URL:  request.Link,
			Tags: request.Tags,
		}, nil
	})

	dispatcher := NewDispatcher([]Handler{trackHandler})
	chatID := int64(1)

	// act
	gotStart := dispatcher.Dispatch(ctx, newCommandMessage(chatID, "/track"))
	gotLink := dispatcher.Dispatch(ctx, newTextMessage(chatID, "https://github.com/user/repo"))
	gotFinish := dispatcher.Dispatch(ctx, newCommandMessage(chatID, "/cancel"))
	dialogAfterFinish := trackHandler.States().Get(chatID)

	// assert
	wantStart := "Отправьте ссылку для отслеживания."
	if gotStart != wantStart {
		t.Errorf("unexpected response on /track:\nwant: %q\ngot:  %q", wantStart, gotStart)
	}

	wantLink := "Введите теги через запятую, введите /cancel, чтобы не добавлять теги."
	if gotLink != wantLink {
		t.Errorf("unexpected response after valid link:\nwant: %q\ngot:  %q", wantLink, gotLink)
	}

	wantFinish := "Ссылка добавлена в отслеживание."
	if gotFinish != wantFinish {
		t.Errorf("unexpected response after /cancel on tags step:\nwant: %q\ngot:  %q", wantFinish, gotFinish)
	}

	if !addCalled {
		t.Errorf("expected addLink to be called")
	}

	if gotChatID != chatID {
		t.Errorf("unexpected chatID in addLink: got %d, want %d", gotChatID, chatID)
	}

	wantReq := scrapperhttp.AddLinkRequest{
		Link: "https://github.com/user/repo",
		Tags: []string{},
	}
	if !reflect.DeepEqual(gotReq, wantReq) {
		t.Errorf("unexpected addLink request:\nwant: %+v\ngot:  %+v", wantReq, gotReq)
	}

	if dialogAfterFinish.State != StateIdle {
		t.Errorf("unexpected state after finish: got %v, want %v", dialogAfterFinish.State, StateIdle)
	}
}

func newTextMessage(chatID int64, text string) *tgbotapi.Message {
	return &tgbotapi.Message{
		Text: text,
		Chat: &tgbotapi.Chat{
			ID: chatID,
		},
	}
}
