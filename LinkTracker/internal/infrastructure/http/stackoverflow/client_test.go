package stackoverflowhttp

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

func TestClient_GetQuestionEvents_ReturnsTimeoutError_WhenServerRespondsTooSlow(t *testing.T) {
	//arrange
	serverDelay := 200 * time.Millisecond
	clientTimeout := 50 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(serverDelay)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[{"title":"test question"}]}`))
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
	_, err := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))
	elapsed := time.Since(start)

	//assert
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if elapsed >= serverDelay {
		t.Errorf("request was not interrupted by timeout: elapsed=%v, server delay=%v", elapsed, serverDelay)
	}
}

func TestClient_GetQuestionEvents_RetriesOnRetryableStatus(t *testing.T) {
	//arrange
	var questionRequestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/questions/123":
			count := atomic.AddInt32(&questionRequestsCount, 1)
			if count < 3 {
				w.WriteHeader(http.StatusBadGateway)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[{"title":"test question"}]}`))
		case "/questions/123/answers":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[],"has_more":false}`))
		case "/questions/123/comments":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[],"has_more":false}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(3, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(100, 100, 1, time.Second),
	)

	//act
	_, err := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))

	//assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := atomic.LoadInt32(&questionRequestsCount); got != 3 {
		t.Errorf("unexpected question requests count: got %d, want 3", got)
	}
}

func TestClient_GetQuestionEvents_DoesNotRetryOnNonRetryableStatus(t *testing.T) {
	//arrange
	var requestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requestsCount, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(3, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(100, 100, 1, time.Second),
	)

	//act
	_, err := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))

	//assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := atomic.LoadInt32(&requestsCount); got != 1 {
		t.Errorf("unexpected requests count: got %d, want 1", got)
	}

	if !strings.Contains(err.Error(), "unexpected status") {
		t.Errorf("unexpected error: got %q", err.Error())
	}
}

func TestClient_GetQuestionEvents_UsesConstantBackoff(t *testing.T) {
	//arrange
	retryDelay := 50 * time.Millisecond
	requestTimes := make([]time.Time, 0, 3)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(3, retryDelay, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(100, 100, 1, time.Second),
	)

	//act
	_, err := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))

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

func TestClient_GetQuestionEvents_CircuitBreakerMovesToOpen(t *testing.T) {
	//arrange
	var requestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requestsCount, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(1, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(2, 50, 1, time.Second),
	)

	//act
	_, firstErr := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))
	_, secondErr := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))
	_, thirdErr := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))

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

func TestClient_GetQuestionEvents_CircuitBreakerMovesFromHalfOpenToClosed(t *testing.T) {
	//arrange
	waitDuration := 30 * time.Millisecond
	var questionRequestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/questions/123":
			count := atomic.AddInt32(&questionRequestsCount, 1)
			if count <= 2 {
				w.WriteHeader(http.StatusBadGateway)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[{"title":"test question"}]}`))
		case "/questions/123/answers":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[],"has_more":false}`))
		case "/questions/123/comments":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[],"has_more":false}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(1, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(2, 50, 1, waitDuration),
	)

	//act
	_, _ = client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))
	_, _ = client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))

	time.Sleep(waitDuration + 10*time.Millisecond)

	_, halfOpenErr := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))
	_, closedErr := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))

	//assert
	if halfOpenErr != nil {
		t.Fatalf("expected half-open trial request to succeed, got %v", halfOpenErr)
	}

	if closedErr != nil {
		t.Fatalf("expected closed circuit breaker request to succeed, got %v", closedErr)
	}

	if got := atomic.LoadInt32(&questionRequestsCount); got != 4 {
		t.Errorf("unexpected question requests count: got %d, want 4", got)
	}
}

func TestClient_GetQuestionEvents_CircuitBreakerMovesFromHalfOpenToOpen(t *testing.T) {
	//arrange
	waitDuration := 30 * time.Millisecond
	var requestsCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requestsCount, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(
		server.URL,
		server.Client(),
		testRetryConfig(1, time.Millisecond, []int{500, 502, 503, 504}),
		testCircuitBreakerConfig(2, 50, 1, waitDuration),
	)

	//act
	_, _ = client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))
	_, _ = client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))

	time.Sleep(waitDuration + 10*time.Millisecond)

	_, halfOpenErr := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))
	_, openErr := client.GetQuestionEvents(context.Background(), 123, time.Now().Add(-time.Hour))

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
