package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dispatcher"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadBotConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tg, err := telegram.NewClient(cfg.AppTelegramToken, cfg.PollTimeoutSeconds)
	if err != nil {
		logger.Error("failed to init telegram client", "error", err)
		os.Exit(1)
	}

	// For communication with scrapper
	httpClient := &http.Client{Timeout: cfg.HttpTimeout}
	scrapperClient := scrapperhttp.NewClient(cfg.ScrapperBaseURL(), httpClient)

	parser := schedulerlink.NewService()

	// Telegram commands dispatcher
	d := dispatcher.NewDispatcher([]dispatcher.Handler{
		dispatcher.NewStart(scrapperClient.RegisterChat),
		dispatcher.NewList(scrapperClient.ListLinks),
		dispatcher.NewTrack(parser, scrapperClient.AddLink),
		dispatcher.NewUntrack(parser, scrapperClient.RemoveLink),
	})

	registerBotCommands(tg, d.Commands(), logger)

	// For communication with scrapper
	httpServer := bothttp.NewServer(cfg.BotAddress(), tg.SendMessage)
	httpServer.Start(logger, stop)

	updates := tg.UpdatesChan(0)

	logger.Info("bot started")

	// Receive and process messages
	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down bot")

			if err := httpServer.Shutdown(context.Background()); err != nil {
				logger.Error("failed to shutdown bot http server", "error", err)
			}

			return

		case u := <-updates:
			logger.Info("update received", "update_id", u.UpdateID)

			if u.Message == nil {
				logger.Info("skip non-message update", "update_id", u.UpdateID)
				continue
			}

			chatID := u.Message.Chat.ID
			logger.Info("processing message",
				"update_id", u.UpdateID,
				"chat_id", chatID,
				"text", u.Message.Text,
			)

			response := d.Dispatch(u.Message)
			if err := tg.SendMessage(chatID, response); err != nil {
				logger.Error("send message failed",
					"error", err,
					"chat_id", chatID,
					"update_id", u.UpdateID,
				)
			}

			logger.Info("response sent",
				"update_id", u.UpdateID,
				"chat_id", chatID,
			)
		}
	}
}
