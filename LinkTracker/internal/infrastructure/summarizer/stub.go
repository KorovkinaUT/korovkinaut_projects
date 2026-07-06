package summarizer

import (
	"context"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
)

type StubSummarizer struct {
	threshold int
}

var _ service.Summarizer = (*StubSummarizer)(nil)

func NewStubSummarizer(threshold int) *StubSummarizer {
	return &StubSummarizer{
		threshold: threshold,
	}
}

func (s *StubSummarizer) Summarize(ctx context.Context, text string) (string, error) {
	text = strings.TrimSpace(text)
	runes := []rune(text)

	if len(runes) <= s.threshold {
		return text, nil
	}

	return string(runes[:s.threshold]) + "...", nil
}
