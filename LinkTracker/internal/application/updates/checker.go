package updates

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	schedulerlink "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/scheduler_link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain/update"
)

// For checking and sending updates.
type Checker struct {
	logger       *slog.Logger
	batchSize    int64
	workersCount int

	subscriptions service.SubscriptionService
	parser        *schedulerlink.Service
	sender        sender.RawUpdateSender

	clients map[schedulerlink.LinkType]LinkClient
}

type trackedLink struct {
	URL         string
	LastUpdated time.Time
}

type checkResult struct {
	url        string
	err        error
	collectErr error
	problem    *sender.Problem
}

func NewChecker(
	logger *slog.Logger,
	batchSize int64,
	workersCount int,
	subscriptions service.SubscriptionService,
	parser *schedulerlink.Service,
	sender sender.RawUpdateSender,
	clients []LinkClient,
) *Checker {
	clientsByType := make(map[schedulerlink.LinkType]LinkClient, len(clients))
	for _, client := range clients {
		clientsByType[client.Type()] = client
	}

	return &Checker{
		logger:        logger,
		batchSize:     batchSize,
		workersCount:  workersCount,
		subscriptions: subscriptions,
		parser:        parser,
		sender:        sender,
		clients:       clientsByType,
	}
}

func (c *Checker) Check(ctx context.Context) error {
	var offset int64
	problems := make([]sender.Problem, 0)

	for {
		batch, err := c.subscriptions.ListTrackedURLs(ctx, c.batchSize, offset)
		if err != nil {
			return fmt.Errorf("list tracked urls batch: %w", err)
		}

		if len(batch) == 0 {
			break
		}

		// make slice from map for parallel processing
		trackedLinks := makeTrackedLinks(batch)

		var batchProblems []sender.Problem
		batchProblems = c.processBatch(ctx, trackedLinks)

		problems = append(problems, batchProblems...)
		offset += int64(len(batch))
	}

	if len(problems) > 0 {
		if err := c.sender.SendProblems(ctx, problems); err != nil {
			c.logger.Error(
				"failed to send problems message",
				"error", err,
			)
		}
	}

	return nil
}

func (c *Checker) checkURL(ctx context.Context, rawURL string, lastUpdated time.Time) error {
	// get basic link info
	parsedLink, err := c.parser.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse tracked url %q: %w", rawURL, err)
	}

	// get client for asking updates
	client, ok := c.clients[parsedLink.Type()]
	if !ok {
		return fmt.Errorf("no client registered for link type %q", parsedLink.Type())
	}

	events, err := client.GetNewEvents(ctx, parsedLink, lastUpdated)
	if err != nil {
		return fmt.Errorf("get events for %q: %w", rawURL, err)
	}

	// GetNewEvents works with precision to the second, so finer filtering is needed
	newEvents := make([]update.Event, 0)
	for _, e := range events {
		if e.CreatedAt().After(lastUpdated) {
			newEvents = append(newEvents, e)
		}
	}
	if len(newEvents) == 0 {
		return nil
	}

	// get subscribed chats
	chatIDs, err := c.subscriptions.ListChatIDsAll(ctx, rawURL)
	if err != nil {
		return fmt.Errorf("list chat ids for url %q: %w", rawURL, err)
	}

	if len(chatIDs) == 0 {
		return nil
	}

	rawEvents, err := makeSenderEvents(parsedLink.Type(), newEvents)
	if err != nil {
		return fmt.Errorf("make sender events for url %q: %w", rawURL, err)
	}

	// send updates to agent
	err = c.sender.SendUpdateEvents(ctx, sender.RawUpdateEvents{
		URL:       rawURL,
		Events:    rawEvents,
		TgChatIDs: chatIDs,
	})
	if err != nil {
		return fmt.Errorf("send raw update events for url %q: %w", rawURL, err)
	}

	// update info about last upodated time in database
	if err := c.subscriptions.UpdateLastUpdated(ctx, rawURL, latestEventTime(newEvents)); err != nil {
		return fmt.Errorf("update last updated for url %q: %w", rawURL, err)
	}

	return nil
}

func (c *Checker) processBatch(ctx context.Context, links []trackedLink) []sender.Problem {
	jobs := make(chan trackedLink, len(links))
	results := make(chan checkResult, len(links))

	var wg sync.WaitGroup

	workersCount := c.workersCount
	if workersCount > len(links) {
		workersCount = len(links)
	}

	// run workers
	for i := 0; i < workersCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for link := range jobs {
				results <- c.processOneLink(ctx, link)
			}
		}()
	}

	// send links to chan
	for _, link := range links {
		jobs <- link
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	// aggregate errors from results
	problems := make([]sender.Problem, 0)
	for result := range results {
		if result.err != nil {
			c.logger.Error(
				"failed to check tracked url",
				"url", result.url,
				"error", result.err,
			)
		}

		if result.collectErr != nil {
			c.logger.Error(
				"failed to collect problems for tracked url",
				"url", result.url,
				"error", result.collectErr,
			)
		}

		if result.problem != nil {
			problems = append(problems, *result.problem)
		}
	}

	return problems
}

func (c *Checker) processOneLink(ctx context.Context, link trackedLink) checkResult {
	err := c.checkURL(ctx, link.URL, link.LastUpdated)
	if err == nil {
		return checkResult{
			url: link.URL,
		}
	}

	p, collectErr := c.getProblem(ctx, link.URL, err)

	return checkResult{
		url:        link.URL,
		err:        err,
		collectErr: collectErr,
		problem:    p,
	}
}

func makeTrackedLinks(batch map[string]time.Time) []trackedLink {
	result := make([]trackedLink, 0, len(batch))
	for url, lastUpdated := range batch {
		result = append(result, trackedLink{
			URL:         url,
			LastUpdated: lastUpdated,
		})
	}

	return result
}

func (c *Checker) getProblem(ctx context.Context, rawURL string, checkErr error) (*sender.Problem, error) {
	chatIDs, err := c.subscriptions.ListChatIDsAll(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("list chat ids for problem url %q: %w", rawURL, err)
	}

	if len(chatIDs) == 0 {
		return nil, nil
	}

	return &sender.Problem{
		URL:     rawURL,
		Message: checkErr.Error(),
		ChatIDs: chatIDs,
	}, nil
}

func latestEventTime(events []update.Event) time.Time {
	latest := events[0].CreatedAt()

	for _, e := range events[1:] {
		if e.CreatedAt().After(latest) {
			latest = e.CreatedAt()
		}
	}

	return latest
}

func makeSenderEvents(linkType schedulerlink.LinkType, events []update.Event) ([]sender.Event, error) {
	result := make([]sender.Event, 0, len(events))

	for _, event := range events {
		result = append(result, sender.Event{
			Source:       string(linkType),
			Type:         strings.TrimSpace(event.EventType()),
			Title:        strings.TrimSpace(event.EventTitle()),
			Author:       strings.TrimSpace(event.EventAuthor()),
			CreationTime: event.CreatedAt(),
			Preview:      strings.TrimSpace(event.EventPreview()),
		})
	}

	return result, nil
}
