package dispatcher

import (
	"math/rand"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type mockCommand struct{}

func (mockCommand) Command() string     { return "cmd" }
func (mockCommand) Description() string { return "descr" }
func (mockCommand) Handle(msg *tgbotapi.Message) string {
	return "resp"
}

type mockHandler struct {
	command     string
	description string
	response    string
	calls       *int
}

func (h mockHandler) Command() string     { return h.command }
func (h mockHandler) Description() string { return h.description }
func (h mockHandler) Handle(msg *tgbotapi.Message) string {
	(*h.calls)++
	return h.response
}

func TestDispatcher_Dispatch_Start(t *testing.T) {
	startCalls := 0
	otherCalls := 0

	dispatcher := NewDispatcher([]Handler{
		mockHandler{
			command:     "start",
			description: "start command",
			response:    "start response",
			calls:       &startCalls,
		},
		mockHandler{
			command:     "other",
			description: "other command",
			response:    "other response",
			calls:       &otherCalls,
		},
	})

	msg := newCommandMessage(1, "/start")

	got := dispatcher.Dispatch(msg)

	want := "start response"

	if got != want {
		t.Errorf("unexpected response: got %q, want %q", got, want)
	}

	if startCalls != 1 {
		t.Errorf("start handler must be called once, got %d", startCalls)
	}

	if otherCalls != 0 {
		t.Errorf("other handler must not be called, got %d", otherCalls)
	}
}

func TestDispatcher_Dispatch_Help(t *testing.T) {
	dispatcher := NewDispatcher([]Handler{
		mockCommand{},
	})

	msg := newCommandMessage(1, "/help")

	got := dispatcher.Dispatch(msg)

	if !strings.Contains(got, "/cmd - descr") {
		t.Errorf("help response must contain mock command description, got %q", got)
	}
}

func TestDispatcher_Dispatch_UnknownCommand(t *testing.T) {
	dispatcher := NewDispatcher([]Handler{
		mockHandler{
			command:     "start",
			description: "start command",
			response:    "start response",
			calls:       new(int),
		},
	})

	want := unknownCommandMsg

	cmd := randomString(10)
	commandMsg := newCommandMessage(1, "/"+cmd)

	text := randomString(11)
	textMsg := &tgbotapi.Message{
		Text: text,
		Chat: &tgbotapi.Chat{
			ID: 1,
		},
	}

	gotCommand := dispatcher.Dispatch(commandMsg)
	gotText := dispatcher.Dispatch(textMsg)

	if gotCommand != want {
		t.Errorf("unexpected response for unknown command: got %q, want %q", gotCommand, want)
	}

	if gotText != want {
		t.Errorf("unexpected response for unknown text: got %q, want %q", gotText, want)
	}
}

var rnd = rand.New(rand.NewSource(1))

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"

	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}
	return string(b)
}