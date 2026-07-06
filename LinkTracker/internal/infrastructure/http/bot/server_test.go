package bothttp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http"
)

func newTestServer(rateLimitCfg *config.RateLimitConfig) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("/updates", NewUpdatesHandler(func(chatID int64, text string) error {
		return nil
	}))

	var handler http.Handler = mux
	if rateLimitCfg != nil {
		handler = httpinfra.NewRateLimiter(rateLimitCfg).Middleware(handler)
	}

	return httptest.NewServer(handler)
}

func TestServer_UpdatesEndpoint_ValidRequest(t *testing.T) {
	//arrange
	ts := newTestServer(nil)
	defer ts.Close()

	body := []byte(`{
		"id": 1,
		"url": "https://github.com/user/repo",
		"description": "Link was updated",
		"tgChatIds": [1, 2]
	}`)

	//act
	resp, err := http.Post(ts.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	//assert
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestServer_UpdatesEndpoint_MissingRequeiredField(t *testing.T) {
	//arrange
	ts := newTestServer(nil)
	defer ts.Close()

	body := []byte(`{
		"id": 1,
		"url": "https://github.com/user/repo"
	}`)

	//act
	resp, err := http.Post(ts.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	//assert
	if resp.StatusCode == http.StatusOK {
		t.Errorf("unexpected status code: got %d, want non-%d", resp.StatusCode, http.StatusOK)
	}
}

func TestServer_UpdatesEndpoint_InvalidFieldType(t *testing.T) {
	//arrange
	ts := newTestServer(nil)
	defer ts.Close()

	body := []byte(`{
		"id": "wrong",
		"url": "https://github.com/user/repo",
		"description": "Link was updated",
		"tgChatIds": [1, 2]
	}`)

	//act
	resp, err := http.Post(ts.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	//assert
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unexpected status code: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestServer_UpdatesEndpoint_ReturnsTooManyRequests_WhenRateLimitExceeded(t *testing.T) {
	//arrange
	ts := newTestServer(&config.RateLimitConfig{
		RPS:   0.001,
		Burst: 1,
	})
	defer ts.Close()

	body := []byte(`{
		"id": 1,
		"url": "https://github.com/user/repo",
		"description": "Link was updated",
		"tgChatIds": [1, 2]
	}`)

	//act
	firstResp, err := http.Post(ts.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send first request: %v", err)
	}
	defer firstResp.Body.Close()

	secondResp, err := http.Post(ts.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send second request: %v", err)
	}
	defer secondResp.Body.Close()

	//assert
	if firstResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected first status code: got %d, want %d", firstResp.StatusCode, http.StatusOK)
	}

	if secondResp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("unexpected second status code: got %d, want %d", secondResp.StatusCode, http.StatusTooManyRequests)
	}
}
