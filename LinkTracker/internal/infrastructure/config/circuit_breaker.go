package config

import (
	"fmt"
	"time"
)

type CircuitBreakerConfig struct {
	SlidingWindowSize       uint32        `envconfig:"HTTP_CB_SLIDING_WINDOW_SIZE" default:"10"`
	MinimumRequiredCalls    uint32        `envconfig:"HTTP_CB_MINIMUM_REQUIRED_CALLS" default:"5"`
	FailureRateThreshold    float64       `envconfig:"HTTP_CB_FAILURE_RATE_THRESHOLD" default:"50"`
	CallsInHalfOpen         uint32        `envconfig:"HTTP_CB_CALLS_IN_HALF_OPEN" default:"5"`
	WaitDurationInOpenState time.Duration `envconfig:"HTTP_CB_WAIT_DURATION_IN_OPEN_STATE" default:"1s"`
}

func (c *CircuitBreakerConfig) Validate() error {
	if c.SlidingWindowSize == 0 {
		c.SlidingWindowSize = 10
	}

	if c.MinimumRequiredCalls == 0 {
		c.MinimumRequiredCalls = 5
	}

	if c.MinimumRequiredCalls > c.SlidingWindowSize {
		return fmt.Errorf(
			"minimum required calls %d must be less than or equal to sliding window size %d",
			c.MinimumRequiredCalls,
			c.SlidingWindowSize,
		)
	}

	if c.FailureRateThreshold <= 0 {
		c.FailureRateThreshold = 50
	}

	if c.FailureRateThreshold > 100 {
		return fmt.Errorf("failure rate threshold %.2f must be less than or equal to 100", c.FailureRateThreshold)
	}

	if c.CallsInHalfOpen == 0 {
		c.CallsInHalfOpen = 5
	}

	if c.WaitDurationInOpenState <= 0 {
		c.WaitDurationInOpenState = time.Second
	}

	return nil
}
