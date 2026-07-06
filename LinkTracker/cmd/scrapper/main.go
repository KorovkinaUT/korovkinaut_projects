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
	githubhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/github"
	scrapperhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/scrapper"
	stackoverflowhttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/stackoverflow"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/scheduler"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/sender"
	valkeycache "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/valkey"
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

	kafkaCfg, err := config.LoadKafkaConfig()
	if err != nil {
		logger.Error("failed to load kafka config", "error", err)
		os.Exit(1)
	}

	var valkeyCfg *config.ValkeyConfig
	if cfg.CacheEnabled {
		valkeyCfg, err = config.LoadValkeyConfig()
		if err != nil {
			logger.Error("failed to load valkey config", "error", err)
		}
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

	// For GET /list requests caching
	var listCache service.ListCache
	if cfg.CacheEnabled && valkeyCfg != nil {
		valkeyClient, err := valkeycache.NewClient(valkeyCfg)
		if err != nil {
			logger.Error("failed to create valkey client", "error", err)
		} else {
			defer valkeyClient.Close()

			listCache = valkeycache.NewListCache(
				valkeyClient,
				valkeyCfg.TTL,
				valkeyCfg.ClientSideCacheTTL,
				valkeyCfg.Timeout,
			)
		}
	}

	// Stores subscriptions and chats info
	subscriptionService := service.NewSubscriptionService(
		cfg.CacheEnabled,
		chatRepository,
		subscriptionRepository,
		listCache,
		logger,
	)

	// For communication with bot
	httpServer := scrapperhttp.NewServer(cfg.ScrapperAddress(), subscriptionService, cfg.RateLimit)
	httpServer.Start(logger, stop)

	// For communication with sites and bot
	httpClient := &http.Client{Timeout: cfg.HttpTimeout}
	githubClient := githubhttp.NewClient(cfg.GithubBaseURL, httpClient, cfg.Retry, cfg.CircuitBreaker)
	stackClient := stackoverflowhttp.NewClient(cfg.StackOverflowBaseURL, httpClient, cfg.Retry, cfg.CircuitBreaker)
	updatesSender := sender.NewAgentKafkaSender(kafkaCfg.Brokers, kafkaCfg.RawUpdatesTopic)

	// Parser for Checker of updates
	parser := schedulerlink.NewService()

	// Clinet interfaces for Checker
	githubLinkClient := updates.NewGitHubClient(githubClient)
	stackOverflowLinkClient := updates.NewStackOverflowClient(stackClient)

	checker := updates.NewChecker(
		logger,
		cfg.BatchSize,
		cfg.WorkersCount,
		subscriptionService,
		parser,
		updatesSender,
		[]updates.LinkClient{
			githubLinkClient,
			stackOverflowLinkClient,
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

	if err := updatesSender.Close(); err != nil {
		logger.Error("failed to close message sender", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown scrapper http server", "error", err)
	}
}
