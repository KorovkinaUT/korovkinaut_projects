package dispatcher

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const unknownCommandMsg = "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд."

// Interface for commands
type Handler interface {
	Command() string
	Description() string
	Handle(msg *tgbotapi.Message) string
}

// Interface for /track
type dialogHandler interface {
	States() *StateStorage
	HandleDialog(msg *tgbotapi.Message, dialog TrackDialog) string
}

// Commands dispatcher
type Dispatcher struct {
	handlers map[string]Handler
	helpText string
}

func NewDispatcher(h []Handler) *Dispatcher {
	m := make(map[string]Handler, len(h)+1)
	descriptions := make([]string, 0, len(h)+1)

	helpHandler := help{}
	m[helpHandler.Command()] = helpHandler
	descriptions = append(descriptions, "/"+helpHandler.Command()+" - "+helpHandler.Description())

	for _, handler := range h {
		m[handler.Command()] = handler
		descriptions = append(descriptions, "/"+handler.Command()+" - "+handler.Description())
	}

	return &Dispatcher{
		handlers: m,
		helpText: strings.Join(descriptions, "\n"),
	}
}

func (d *Dispatcher) Dispatch(msg *tgbotapi.Message) string {
	if msg == nil || msg.Chat == nil {
		return "Не удалось определить чат."
	}

	track, ok := d.handlers["track"]
	// If /track command is added
	if ok {
		trackHandler, ok := track.(dialogHandler)
		if !ok {
			return "Не удалось обработать состояние диалога."
		}

		dialog := trackHandler.States().Get(msg.Chat.ID)
		if dialog.State != StateIdle {
			// Continue dialog
			if !msg.IsCommand() {
				return trackHandler.HandleDialog(msg, dialog)
			}

			if msg.Command() == "cancel" {
				return trackHandler.HandleDialog(msg, dialog)
			}

			// If send other command, cancel tracking and handle command
			trackHandler.States().Reset(msg.Chat.ID)
		}
	}

	if !msg.IsCommand() {
		return unknownCommandMsg
	}

	cmd := msg.Command()

	if cmd == "help" {
		return d.helpText
	}

	h, ok := d.handlers[cmd]
	if !ok {
		return unknownCommandMsg
	}

	return h.Handle(msg)
}

func (d *Dispatcher) Commands() []Handler {
	cmds := make([]Handler, 0, len(d.handlers))
	for _, h := range d.handlers {
		cmds = append(cmds, h)
	}
	return cmds
}

func (d *Dispatcher) AddHandler(h Handler) {

}
