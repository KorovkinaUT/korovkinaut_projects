package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/updates"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
	githubhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/github"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
	stackoverflowhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/stackoverflow"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/memory"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/scheduler"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadScrapperConfig()
	if err != nil {
		logger.Error("failed to load scrapper config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	chatRepository := memory.NewChatRepository()
	subscriptionRepository := memory.NewSubscriptionRepository()

	// Stores subscriptions and chats info
	subscriptionService := service.NewSubscriptionService(
		chatRepository,
		subscriptionRepository,
	)

	// For communication with bot
	httpServer := scrapperhttp.NewServer(cfg.ScrapperAddress(), subscriptionService)
	httpServer.Start(logger, stop)

	// For communication with sites and bot
	httpClient := &http.Client{Timeout: cfg.HttpTimeout}
	githubClient := githubhttp.NewClient(cfg.GithubBaseURL, httpClient)
	stackClient := stackoverflowhttp.NewClient(cfg.StackOverflowBaseURL, httpClient)
	botClient := bothttp.NewClient(cfg.BotBaseURL(), httpClient)

	// Parser for Checker of updates
	parser := schedulerlink.NewService()
	
	// Clinet interfaces for Checker
	githubLinkClient := updates.NewGitHubClient(githubClient)
	stackOverflowLinkClient := updates.NewStackOverflowClient(stackClient)

	checker := updates.NewChecker(
		logger,
		subscriptionService,
		parser,
		botClient,
		githubLinkClient,
		stackOverflowLinkClient,
	)

	// Checking updates job
	updatesJob := updates.NewJob(logger, checker)

	// Scheduler of checking updates job
	scrapperScheduler, err := scheduler.New(
		updatesJob,
		cfg.SchedulerInterval,
		logger,
	)
	if err != nil {
		logger.Error("failed to create scheduler", "error", err)
		os.Exit(1)
	}

	if err := scrapperScheduler.Start(ctx); err != nil {
		logger.Error("failed to start scheduler", "error", err)
		os.Exit(1)
	}

	logger.Info("scrapper started")

	<-ctx.Done()

	logger.Info("shutting down scrapper")

	if err := scrapperScheduler.Stop(); err != nil {
		logger.Error("failed to stop scheduler", "error", err)
	}

	if err := httpServer.Shutdown(context.Background()); err != nil {
		logger.Error("failed to shutdown scrapper http server", "error", err)
	}
}
