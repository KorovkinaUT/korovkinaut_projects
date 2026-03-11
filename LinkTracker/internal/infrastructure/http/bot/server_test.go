package bothttp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("/updates", NewUpdatesHandler(func(chatID int64, text string) error {
		return nil
	}))
	return httptest.NewServer(mux)
}

func TestServer_UpdatesEndpoint_ValidRequest(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	body := []byte(`{
		"id": 1,
		"url": "https://github.com/user/repo",
		"description": "Link was updated",
		"tgChatIds": [1, 2]
	}`)

	resp, err := http.Post(ts.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestServer_UpdatesEndpoint_MissingRequeiredField(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	body := []byte(`{
		"id": 1,
		"url": "https://github.com/user/repo"
	}`)

	resp, err := http.Post(ts.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want non-%d", resp.StatusCode, http.StatusOK)
	}
}

func TestServer_UpdatesEndpoint_InvalidFieldType(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	body := []byte(`{
		"id": "wrong",
		"url": "https://github.com/user/repo",
		"description": "Link was updated",
		"tgChatIds": [1, 2]
	}`)

	resp, err := http.Post(ts.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
