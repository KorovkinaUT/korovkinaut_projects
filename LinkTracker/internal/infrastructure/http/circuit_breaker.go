package httpinfra

import (
	"fmt"

	"github.com/sony/gobreaker"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func NewCircuitBreaker(name string, cfg *config.CircuitBreakerConfig) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: cfg.CallsInHalfOpen,
		Timeout:     cfg.WaitDurationInOpenState,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < cfg.SlidingWindowSize {
				return false
			}

			failureRate := float64(counts.TotalFailures) / float64(counts.Requests) * 100

			return failureRate >= cfg.FailureRateThreshold
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			fmt.Printf("circuit breaker %s changed state from %s to %s\n", name, from, to)
		},
	})
}
