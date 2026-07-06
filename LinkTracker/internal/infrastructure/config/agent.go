package config

import (
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

type AgentConfig struct {
	StopWords         []string `envconfig:"AGENT_STOP_WORDS" default:"advertisement,promo"`
	ExcludedAuthors   []string `envconfig:"AGENT_EXCLUDED_AUTHORS" default:""`
	MinLength         int      `envconfig:"AGENT_MIN_LENGTH" default:"10"`
	SummaryThreshold  int      `envconfig:"AGENT_SUMMARY_THRESHOLD" default:"100"`
	SummarizationMode string   `envconfig:"AGENT_SUMMARIZATION_MODE" default:"STUB"`
}

func LoadAgentConfig() (*AgentConfig, error) {
	var cfg AgentConfig
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	cfg.SummarizationMode = strings.ToUpper(cfg.SummarizationMode)

	if cfg.MinLength < 0 {
		cfg.MinLength = 10
	}
	if cfg.SummaryThreshold <= 0 {
		cfg.SummaryThreshold = 100
	}

	switch cfg.SummarizationMode {
	case "STUB", "AI":
		return &cfg, nil
	default:
		return nil, fmt.Errorf(
			"unsupported AI_AGENT_SUMMARIZATION_MODE %q: expected one of [STUB, AI]",
			cfg.SummarizationMode,
		)
	}
}
