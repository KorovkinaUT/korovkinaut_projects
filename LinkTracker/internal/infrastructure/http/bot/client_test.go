package bothttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

func TestClient_SendUpdate_ReturnsTimeoutError_WhenServerRespondsTooSlow(t *testing.T) {
	//arrange
	serverDelay := 200 * time.Millisecond
	clientTimeout := 50 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(serverDelay)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		&http.Client{Timeout: clientTimeout},
		testRetryConfig(3, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(100, 100, 1, time.Second),
	)

	//act
	start := time.Now()
	err := client.SendUpdate(context.Background(), testLinkUpdate())
	elapsed := time.Since(start)

	//assert
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if elapsed >= serverDelay {
		t.Errorf("request was not interrupted by timeout: elapsed=%v, server delay=%v", elapsed, serverDelay)
	}
}

func TestClient_SendUpdate_RetriesOnRetryableStatus(t *testing.T) {
	//arrange
	var requestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := atomic.AddInt32(&requestsCount, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(3, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(100, 100, 1, time.Second),
	)

	//act
	err := client.SendUpdate(context.Background(), testLinkUpdate())

	//assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := atomic.LoadInt32(&requestsCount); got != 3 {
		t.Errorf("unexpected requests count: got %d, want 3", got)
	}
}

func TestClient_SendUpdate_DoesNotRetryOnNonRetryableStatus(t *testing.T) {
	//arrange
	var requestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requestsCount, 1)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(3, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(100, 100, 1, time.Second),
	)

	//act
	err := client.SendUpdate(context.Background(), testLinkUpdate())

	//assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := atomic.LoadInt32(&requestsCount); got != 1 {
		t.Errorf("unexpected requests count: got %d, want 1", got)
	}

	if !strings.Contains(err.Error(), "bot api error") {
		t.Errorf("unexpected error: got %q", err.Error())
	}
}

func TestClient_SendUpdate_UsesConstantBackoff(t *testing.T) {
	//arrange
	retryDelay := 50 * time.Millisecond
	requestTimes := make([]time.Time, 0, 3)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(3, retryDelay, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(100, 100, 1, time.Second),
	)

	//act
	err := client.SendUpdate(context.Background(), testLinkUpdate())

	//assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(requestTimes) != 3 {
		t.Fatalf("unexpected requests count: got %d, want 3", len(requestTimes))
	}

	firstDelay := requestTimes[1].Sub(requestTimes[0])
	secondDelay := requestTimes[2].Sub(requestTimes[1])

	if firstDelay < retryDelay {
		t.Errorf("first retry delay is too small: got %v, want at least %v", firstDelay, retryDelay)
	}

	if secondDelay < retryDelay {
		t.Errorf("second retry delay is too small: got %v, want at least %v", secondDelay, retryDelay)
	}
}

func TestClient_SendUpdate_CircuitBreakerMovesToOpen(t *testing.T) {
	//arrange
	var requestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requestsCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(1, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(2, 50, 1, time.Second),
	)

	//act
	firstErr := client.SendUpdate(context.Background(), testLinkUpdate())
	secondErr := client.SendUpdate(context.Background(), testLinkUpdate())
	thirdErr := client.SendUpdate(context.Background(), testLinkUpdate())

	//assert
	if firstErr == nil {
		t.Error("expected first request error, got nil")
	}

	if secondErr == nil {
		t.Error("expected second request error, got nil")
	}

	if !errors.Is(thirdErr, gobreaker.ErrOpenState) {
		t.Errorf("expected open circuit breaker error, got %v", thirdErr)
	}

	if got := atomic.LoadInt32(&requestsCount); got != 2 {
		t.Errorf("unexpected requests count: got %d, want 2", got)
	}
}

func TestClient_SendUpdate_CircuitBreakerMovesFromHalfOpenToClosed(t *testing.T) {
	//arrange
	waitDuration := 30 * time.Millisecond
	var requestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := atomic.AddInt32(&requestsCount, 1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(1, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(2, 50, 1, waitDuration),
	)

	//act
	_ = client.SendUpdate(context.Background(), testLinkUpdate())
	_ = client.SendUpdate(context.Background(), testLinkUpdate())

	time.Sleep(waitDuration + 10*time.Millisecond)

	halfOpenErr := client.SendUpdate(context.Background(), testLinkUpdate())
	closedErr := client.SendUpdate(context.Background(), testLinkUpdate())

	//assert
	if halfOpenErr != nil {
		t.Fatalf("expected half-open trial request to succeed, got %v", halfOpenErr)
	}

	if closedErr != nil {
		t.Fatalf("expected closed circuit breaker request to succeed, got %v", closedErr)
	}

	if got := atomic.LoadInt32(&requestsCount); got != 4 {
		t.Errorf("unexpected requests count: got %d, want 4", got)
	}
}

func TestClient_SendUpdate_CircuitBreakerMovesFromHalfOpenToOpen(t *testing.T) {
	//arrange
	waitDuration := 30 * time.Millisecond
	var requestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requestsCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(1, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(2, 50, 1, waitDuration),
	)

	//act
	_ = client.SendUpdate(context.Background(), testLinkUpdate())
	_ = client.SendUpdate(context.Background(), testLinkUpdate())

	time.Sleep(waitDuration + 10*time.Millisecond)

	halfOpenErr := client.SendUpdate(context.Background(), testLinkUpdate())
	openErr := client.SendUpdate(context.Background(), testLinkUpdate())

	//assert
	if halfOpenErr == nil {
		t.Error("expected half-open trial request to fail, got nil")
	}

	if !errors.Is(openErr, gobreaker.ErrOpenState) {
		t.Errorf("expected open circuit breaker error, got %v", openErr)
	}

	if got := atomic.LoadInt32(&requestsCount); got != 3 {
		t.Errorf("unexpected requests count: got %d, want 3", got)
	}
}

func testLinkUpdate() LinkUpdate {
	return LinkUpdate{
		ID:          1,
		URL:         "https://github.com/user/repo",
		Description: "test update",
		TgChatIDs:   []int64{123},
	}
}

func testRetryConfig(attempts uint, delay time.Duration, statuses []int) *config.RetryConfig {
	return &config.RetryConfig{
		Attempts:          attempts,
		Delay:             delay,
		RetryableStatuses: statuses,
	}
}

func testCircuitBreakerConfig(
	slidingWindowSize uint32,
	failureRateThreshold float64,
	callsInHalfOpen uint32,
	waitDurationInOpenState time.Duration,
) *config.CircuitBreakerConfig {
	return &config.CircuitBreakerConfig{
		SlidingWindowSize:       slidingWindowSize,
		FailureRateThreshold:    failureRateThreshold,
		CallsInHalfOpen:         callsInHalfOpen,
		WaitDurationInOpenState: waitDurationInOpenState,
	}
}