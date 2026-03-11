package schedulerlink

import (
	"fmt"
	"net/url"
	"strings"
)

// Interface for parsers of different sites
type Parser interface {
	Parse(raw string) (SchedulerLink, error)
}

// Service chooses the correct parser for each link and parse it
type Service struct {
	parsers map[string]Parser
}

func NewService() *Service {
	return &Service{
		parsers: map[string]Parser{
			"github.com":           GitHubParser{},
			"stackoverflow.com":    StackOverflowParser{},
		},
	}
}

func (s *Service) Parse(raw string) (SchedulerLink, error) {
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse tracked link url: %w", err)
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme: %s", parsedURL.Scheme)
	}

	host := strings.ToLower(parsedURL.Host)
	parser, ok := s.parsers[host]
	if !ok {
		return nil, fmt.Errorf("unsupported host: %s", parsedURL.Host)
	}

	return parser.Parse(raw)
}