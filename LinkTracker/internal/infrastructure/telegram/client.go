package telegram

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Client wraps Telegram SDK
type Client struct {
	bot     *tgbotapi.BotAPI
	timeout int
}

func NewClient(token string, timeout int) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("init telegram bot: %w", err)
	}
	return &Client{bot: bot, timeout: timeout}, nil
}

func (c *Client) GetUpdates(offset int) ([]tgbotapi.Update, error) {
	cfg := tgbotapi.NewUpdate(offset)
	cfg.Timeout = c.timeout

	updates, err := c.bot.GetUpdates(cfg)
	if err != nil {
		return nil, fmt.Errorf("get updates: %w", err)
	}
	return updates, nil
}

func (c *Client) UpdatesChan(offset int) tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(offset)
	u.Timeout = c.timeout
	return c.bot.GetUpdatesChan(u)
}

func (c *Client) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)

	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

// SetMyCommands configures the command menu shown in Telegram
func (c *Client) SetMyCommands(commands map[string]string) error {
	cmds := make([]tgbotapi.BotCommand, 0, len(commands))
	for name, desc := range commands {
		cmds = append(cmds, tgbotapi.BotCommand{
			Command:     name,
			Description: desc,
		})
	}

	_, err := c.bot.Request(tgbotapi.NewSetMyCommands(cmds...))
	if err != nil {
		return fmt.Errorf("set my commands: %w", err)
	}
	return nil
}
