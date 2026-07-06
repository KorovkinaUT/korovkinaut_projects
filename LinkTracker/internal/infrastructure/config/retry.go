package config

import (
	"fmt"
	"slices"
	"time"
)

type RetryConfig struct {
	Attempts          uint          `envconfig:"HTTP_RETRY_ATTEMPTS" default:"3"`
	Delay             time.Duration `envconfig:"HTTP_RETRY_DELAY" default:"200ms"`
	RetryableStatuses []int         `envconfig:"HTTP_RETRYABLE_STATUSES" default:"500,502,503,504"`
}

func (c *RetryConfig) Validate() error {
	if c.Attempts == 0 {
		c.Attempts = 3
	}

	if c.Delay < 0 {
		c.Delay = 200 * time.Millisecond
	}

	for _, status := range c.RetryableStatuses {
		if status < 100 || status > 599 {
			return fmt.Errorf("invalid retryable status %d", status)
		}
	}

	return nil
}

func (c *RetryConfig) IsRetryableStatus(status int) bool {
	return slices.Contains(c.RetryableStatuses, status)
}
