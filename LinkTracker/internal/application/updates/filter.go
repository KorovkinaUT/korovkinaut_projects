package updates

import (
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

type Filter struct {
	minLength       int
	stopWords       []string
	excludedAuthors map[string]struct{}
}

func NewFilter(cfg *config.AgentConfig) *Filter {
	stopWords := make([]string, 0, len(cfg.StopWords))
	for _, word := range cfg.StopWords {
		word = strings.ToLower(strings.TrimSpace(word))
		if word != "" {
			stopWords = append(stopWords, word)
		}
	}

	excludedAuthors := make(map[string]struct{}, len(cfg.ExcludedAuthors))
	for _, author := range cfg.ExcludedAuthors {
		author = strings.ToLower(strings.TrimSpace(author))
		if author != "" {
			excludedAuthors[author] = struct{}{}
		}
	}

	return &Filter{
		minLength:       cfg.MinLength,
		stopWords:       stopWords,
		excludedAuthors: excludedAuthors,
	}
}

func (f *Filter) ApplyEvents(events []sender.Event) []sender.Event {
	filtered := make([]sender.Event, 0, len(events))

	for _, event := range events {
		if f.allowedEvents(event) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

func (f *Filter) ApplyProblems(problems []sender.Problem) []sender.Problem {
	filtered := make([]sender.Problem, 0, len(problems))

	for _, problem := range problems {
		if f.allowedProblem(problem) {
			filtered = append(filtered, problem)
		}
	}

	return filtered
}

func (f *Filter) allowedEvents(event sender.Event) bool {
	if len([]rune(event.Preview)) < f.minLength {
		return false
	}

	author := strings.ToLower(strings.TrimSpace(event.Author))
	if _, ok := f.excludedAuthors[author]; ok {
		return false
	}

	preview := strings.ToLower(event.Preview)
	for _, word := range f.stopWords {
		if strings.Contains(preview, word) {
			return false
		}
	}

	return true
}

func (f *Filter) allowedProblem(problem sender.Problem) bool {
	if len([]rune(problem.Message)) < f.minLength {
		return false
	}

	message := strings.ToLower(problem.Message)
	for _, word := range f.stopWords {
		if strings.Contains(message, word) {
			return false
		}
	}

	return true
}
