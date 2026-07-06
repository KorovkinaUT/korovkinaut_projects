package summarizer

import (
	"fmt"
	"strings"

	appservice "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func NewSummarizer(cfg *config.AgentConfig) (appservice.Summarizer, error) {
	switch strings.ToUpper(cfg.SummarizationMode) {
	case "STUB":
		return NewStubSummarizer(cfg.SummaryThreshold), nil
	default:
		return nil, fmt.Errorf(
			"unsupported AGENT_SUMMARIZATION_MODE %q: expected one of [STUB, AI]",
			cfg.SummarizationMode,
		)
	}
}
