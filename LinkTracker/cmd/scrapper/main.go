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
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/database"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
	githubhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/github"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
	stackoverflowhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/stackoverflow"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/scheduler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadScrapperConfig()
	if err != nil {
		logger.Error("failed to load scrapper config", "error", err)
		os.Exit(1)
	}

	dbCfg, err := config.LoadDatabaseConfig()
	if err != nil {
		logger.Error("failed to load database config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	poolCtx, cancel := context.WithTimeout(ctx, dbCfg.ConnectTimeout)
	defer cancel()

	// For communication with database
	dbPool, err := database.NewPool(poolCtx, dbCfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	chatRepository, subscriptionRepository, err := database.NewRepositories(dbCfg, dbPool)
	if err != nil {
		logger.Error("failed to initialize repositories", "error", err)
		os.Exit(1)
	}

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
	httpSender := sender.NewHTTPSender(botClient)

	// Parser for Checker of updates
	parser := schedulerlink.NewService()

	// Clinet interfaces for Checker
	githubLinkClient := updates.NewGitHubClient(githubClient)
	stackOverflowLinkClient := updates.NewStackOverflowClient(stackClient)

	// Interfaces for formatting update messages
	githubFormatter := updates.GitHubFormatter{}
	stackOverflowFormatter := updates.StackOverflowFormatter{}

	checker := updates.NewChecker(
		logger,
		cfg.BatchSize,
		cfg.WorkersCount,
		subscriptionService,
		parser,
		httpSender,
		[]updates.LinkClient{
			githubLinkClient,
			stackOverflowLinkClient,
		},
		[]updates.Formatter{
			githubFormatter,
			stackOverflowFormatter,
		},
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown scrapper http server", "error", err)
	}
}
