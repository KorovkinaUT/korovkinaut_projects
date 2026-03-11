package updates

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	bothttp "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/bot"
)

// For checking and sending updates
type Checker struct {
	logger        *slog.Logger
	subscriptions *service.SubscriptionService
	parser        *schedulerlink.Service
	clients       map[schedulerlink.LinkType]LinkClient
	botClient     *bothttp.Client
}

func NewChecker(
	logger *slog.Logger,
	subscriptions *service.SubscriptionService,
	parser *schedulerlink.Service,
	botClient *bothttp.Client,
	clients ...LinkClient,
) *Checker {
	clientsByType := make(map[schedulerlink.LinkType]LinkClient, len(clients))
	for _, client := range clients {
		clientsByType[client.Type()] = client
	}

	return &Checker{
		logger:        logger,
		subscriptions: subscriptions,
		parser:        parser,
		clients:       clientsByType,
		botClient:     botClient,
	}
}

func (c *Checker) Check(ctx context.Context) error {
	trackedURLs, err := c.subscriptions.ListTrackedURLs()
	if err != nil {
		return fmt.Errorf("list tracked urls: %w", err)
	}

	for rawURL, lastUpdated := range trackedURLs {
		if err := c.checkURL(ctx, rawURL, lastUpdated); err != nil {
			c.logger.Error(
				"failed to check tracked url",
				"url", rawURL,
				"error", err,
			)
			continue
		}
	}

	return nil
}

// Checks last update and sends notification about new update
func (c *Checker) checkURL(ctx context.Context, rawURL string, lastUpdated time.Time) error {
	parsedLink, err := c.parser.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse tracked url %q: %w", rawURL, err)
	}

	client, ok := c.clients[parsedLink.Type()]
	if !ok {
		return fmt.Errorf("no client registered for link type %q", parsedLink.Type())
	}

	updatedAt, err := client.GetUpdatedAt(ctx, parsedLink)
	if err != nil {
		return fmt.Errorf("get updated at for %q: %w", rawURL, err)
	}

	if !updatedAt.After(lastUpdated) {
		return nil
	}

	chatIDs, err := c.subscriptions.ListChatIDs(rawURL)
	if err != nil {
		return fmt.Errorf("list chat ids for url %q: %w", rawURL, err)
	}

	if len(chatIDs) == 0 {
		return nil
	}

	if err := c.botClient.SendUpdate(bothttp.LinkUpdate{
		ID:          0,
		URL:         rawURL,
		Description: "Link was updated",
		TgChatIDs:   chatIDs,
	}); err != nil {
		return fmt.Errorf("send update for url %q: %w", rawURL, err)
	}

	if err := c.subscriptions.UpdateLastUpdated(rawURL, updatedAt); err != nil {
		return fmt.Errorf("update last updated for url %q: %w", rawURL, err)
	}

	return nil
}