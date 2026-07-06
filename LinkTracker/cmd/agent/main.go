package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/updates"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/receiver"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/summarizer"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadAgentConfig()
	if err != nil {
		logger.Error("failed to load agent config", "error", err)
		os.Exit(1)
	}

	kafkaCfg, err := config.LoadKafkaConfig()
	if err != nil {
		logger.Error("failed to load kafka config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// For processing updates
	filter := updates.NewFilter(cfg)
	summarizer, err := summarizer.NewSummarizer(cfg)
	if err != nil {
		logger.Error("failed to initialize summarizer", "error", err)
		os.Exit(1)
	}

	// Interfaces for formatting processed update messages
	githubFormatter := updates.GitHubFormatter{}
	stackOverflowFormatter := updates.StackOverflowFormatter{}

	processingService := service.NewUpdatesProcessingService(
		filter,
		summarizer,
		[]service.Formatter{
			githubFormatter,
			stackOverflowFormatter,
		},
	)

	messageSender := sender.NewBotKafkaSender(
		kafkaCfg.Brokers,
		kafkaCfg.ProcessedUpdatesTopic,
	)

	updatesReceiver := receiver.NewAgentKafkaReceiver(
		*kafkaCfg,
		processingService,
		messageSender,
	)

	updatesReceiver.Start(ctx, logger, stop)

	logger.Info("agent started")

	<-ctx.Done()

	logger.Info("shutting down ai agent")

	if err := updatesReceiver.Shutdown(); err != nil {
		logger.Error("failed to shutdown ai agent receiver", "error", err)
	}

	if err := messageSender.Close(); err != nil {
		logger.Error("failed to close message sender", "error", err)
	}

	logger.Info("ai agent stopped")
}
