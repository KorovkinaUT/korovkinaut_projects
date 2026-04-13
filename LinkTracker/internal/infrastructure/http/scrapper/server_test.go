package scrapperhttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/memory"
)

func TestScrapperServer_AddAndListLink(t *testing.T) {
	// arrange
	ts := newTestScrapperServer()

	registerReq, err := http.NewRequest(http.MethodPost, ts.URL+"/tg-chat/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	addBody, err := json.Marshal(AddLinkRequest{
		Link: "https://github.com/user/repo",
		Tags: []string{"backend", "go"},
	})
	if err != nil {
		t.Fatal(err)
	}

	addReq, err := http.NewRequest(http.MethodPost, ts.URL+"/links", bytes.NewReader(addBody))
	if err != nil {
		t.Fatal(err)
	}
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Tg-Chat-Id", "1")

	listReq, err := http.NewRequest(http.MethodGet, ts.URL+"/links", nil)
	if err != nil {
		t.Fatal(err)
	}
	listReq.Header.Set("Tg-Chat-Id", "1")

	// act
	registerResp, err := http.DefaultClient.Do(registerReq)
	if err != nil {
		t.Fatal(err)
	}
	defer registerResp.Body.Close()

	addResp, err := http.DefaultClient.Do(addReq)
	if err != nil {
		t.Fatal(err)
	}
	defer addResp.Body.Close()

	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()

	var listResult ListLinksResponse
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		t.Fatal(err)
	}

	// assert
	if registerResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected register status: got %d, want %d", registerResp.StatusCode, http.StatusOK)
	}

	if addResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected add link status: got %d, want %d", addResp.StatusCode, http.StatusOK)
	}

	if listResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected list status: got %d, want %d", listResp.StatusCode, http.StatusOK)
	}

	if listResult.Size != 1 {
		t.Errorf("unexpected list size: got %d, want %d", listResult.Size, 1)
	}

	if len(listResult.Links) != 1 {
		t.Errorf("unexpected number of links: got %d, want %d", len(listResult.Links), 1)
	}

	if len(listResult.Links) == 1 {
		if listResult.Links[0].URL != "https://github.com/user/repo" {
			t.Errorf("unexpected url: got %q, want %q", listResult.Links[0].URL, "https://github.com/user/repo")
		}

		if !slices.Equal(listResult.Links[0].Tags, []string{"backend", "go"}) {
			t.Errorf("unexpected tags: got %#v, want %#v", listResult.Links[0].Tags, []string{"backend", "go"})
		}
	}
}

func TestScrapperServer_AddAndRemoveLink(t *testing.T) {
	// arrange
	ts := newTestScrapperServer()

	registerReq, err := http.NewRequest(http.MethodPost, ts.URL+"/tg-chat/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	addBody, err := json.Marshal(AddLinkRequest{
		Link: "https://github.com/user/repo",
		Tags: []string{"backend"},
	})
	if err != nil {
		t.Fatal(err)
	}

	addReq, err := http.NewRequest(http.MethodPost, ts.URL+"/links", bytes.NewReader(addBody))
	if err != nil {
		t.Fatal(err)
	}
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Tg-Chat-Id", "1")

	removeBody, err := json.Marshal(RemoveLinkRequest{
		Link: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatal(err)
	}

	removeReq, err := http.NewRequest(http.MethodDelete, ts.URL+"/links", bytes.NewReader(removeBody))
	if err != nil {
		t.Fatal(err)
	}
	removeReq.Header.Set("Content-Type", "application/json")
	removeReq.Header.Set("Tg-Chat-Id", "1")

	listReq, err := http.NewRequest(http.MethodGet, ts.URL+"/links", nil)
	if err != nil {
		t.Fatal(err)
	}
	listReq.Header.Set("Tg-Chat-Id", "1")

	// act
	registerResp, err := http.DefaultClient.Do(registerReq)
	if err != nil {
		t.Fatal(err)
	}
	defer registerResp.Body.Close()

	addResp, err := http.DefaultClient.Do(addReq)
	if err != nil {
		t.Fatal(err)
	}
	defer addResp.Body.Close()

	removeResp, err := http.DefaultClient.Do(removeReq)
	if err != nil {
		t.Fatal(err)
	}
	defer removeResp.Body.Close()

	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()

	var listResult ListLinksResponse
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		t.Fatal(err)
	}

	// assert
	if registerResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected register status: got %d, want %d", registerResp.StatusCode, http.StatusOK)
	}

	if addResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected add status: got %d, want %d", addResp.StatusCode, http.StatusOK)
	}

	if removeResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected remove status: got %d, want %d", removeResp.StatusCode, http.StatusOK)
	}

	if listResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected list status: got %d, want %d", listResp.StatusCode, http.StatusOK)
	}

	if listResult.Size != 0 {
		t.Errorf("unexpected list size after remove: got %d, want %d", listResult.Size, 0)
	}

	if len(listResult.Links) != 0 {
		t.Errorf("unexpected number of links after remove: got %d, want %d", len(listResult.Links), 0)
	}
}

func TestScrapperServer_RemoveLinkFromNonExistingChat(t *testing.T) {
	// arrange
	ts := newTestScrapperServer()

	registerReq, err := http.NewRequest(http.MethodPost, ts.URL+"/tg-chat/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	addBody, err := json.Marshal(AddLinkRequest{
		Link: "https://github.com/user/repo",
		Tags: []string{"backend"},
	})
	if err != nil {
		t.Fatal(err)
	}

	addReq, err := http.NewRequest(http.MethodPost, ts.URL+"/links", bytes.NewReader(addBody))
	if err != nil {
		t.Fatal(err)
	}
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Tg-Chat-Id", "1")

	removeBody, err := json.Marshal(RemoveLinkRequest{
		Link: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatal(err)
	}

	removeReq, err := http.NewRequest(http.MethodDelete, ts.URL+"/links", bytes.NewReader(removeBody))
	if err != nil {
		t.Fatal(err)
	}
	removeReq.Header.Set("Content-Type", "application/json")
	removeReq.Header.Set("Tg-Chat-Id", "999")

	listReq, err := http.NewRequest(http.MethodGet, ts.URL+"/links", nil)
	if err != nil {
		t.Fatal(err)
	}
	listReq.Header.Set("Tg-Chat-Id", "1")

	// act
	registerResp, err := http.DefaultClient.Do(registerReq)
	if err != nil {
		t.Fatal(err)
	}
	defer registerResp.Body.Close()

	addResp, err := http.DefaultClient.Do(addReq)
	if err != nil {
		t.Fatal(err)
	}
	defer addResp.Body.Close()

	removeResp, err := http.DefaultClient.Do(removeReq)
	if err != nil {
		t.Fatal(err)
	}
	defer removeResp.Body.Close()

	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()

	var listResult ListLinksResponse
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		t.Fatal(err)
	}

	// assert
	if registerResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected register status: got %d, want %d", registerResp.StatusCode, http.StatusOK)
	}

	if addResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected add status: got %d, want %d", addResp.StatusCode, http.StatusOK)
	}

	if removeResp.StatusCode == http.StatusOK {
		t.Errorf("unexpected remove status: got %d, want non-200", removeResp.StatusCode)
	}

	if listResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected list status: got %d, want %d", listResp.StatusCode, http.StatusOK)
	}

	if listResult.Size != 1 {
		t.Errorf("unexpected list size: got %d, want %d", listResult.Size, 1)
	}

	if len(listResult.Links) != 1 {
		t.Errorf("unexpected number of links: got %d, want %d", len(listResult.Links), 1)
	}
}

func TestScrapperServer_AddLinkToNonExistingChat(t *testing.T) {
	// arrange
	ts := newTestScrapperServer()

	registerReq, err := http.NewRequest(http.MethodPost, ts.URL+"/tg-chat/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	addBody, err := json.Marshal(AddLinkRequest{
		Link: "https://github.com/user/repo",
		Tags: []string{"backend"},
	})
	if err != nil {
		t.Fatal(err)
	}

	addReq, err := http.NewRequest(http.MethodPost, ts.URL+"/links", bytes.NewReader(addBody))
	if err != nil {
		t.Fatal(err)
	}
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Tg-Chat-Id", "2")

	// act
	registerResp, err := http.DefaultClient.Do(registerReq)
	if err != nil {
		t.Fatal(err)
	}
	defer registerResp.Body.Close()

	addResp, err := http.DefaultClient.Do(addReq)
	if err != nil {
		t.Fatal(err)
	}
	defer addResp.Body.Close()

	// assert
	if registerResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected register status: got %d, want %d", registerResp.StatusCode, http.StatusOK)
	}

	if addResp.StatusCode == http.StatusOK {
		t.Errorf("unexpected add status: got %d, want non-200", addResp.StatusCode)
	}
}

func TestScrapperServer_WorkWithDeletedChat(t *testing.T) {
	// arrange
	ts := newTestScrapperServer()

	registerReq, err := http.NewRequest(http.MethodPost, ts.URL+"/tg-chat/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	deleteReq, err := http.NewRequest(http.MethodDelete, ts.URL+"/tg-chat/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	addBody, err := json.Marshal(AddLinkRequest{
		Link: "https://github.com/user/repo",
		Tags: []string{"backend"},
	})
	if err != nil {
		t.Fatal(err)
	}

	addReq, err := http.NewRequest(http.MethodPost, ts.URL+"/links", bytes.NewReader(addBody))
	if err != nil {
		t.Fatal(err)
	}
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Tg-Chat-Id", "1")

	// act
	registerResp, err := http.DefaultClient.Do(registerReq)
	if err != nil {
		t.Fatal(err)
	}
	defer registerResp.Body.Close()

	deleteResp, err := http.DefaultClient.Do(deleteReq)
	if err != nil {
		t.Fatal(err)
	}
	defer deleteResp.Body.Close()

	addResp, err := http.DefaultClient.Do(addReq)
	if err != nil {
		t.Fatal(err)
	}
	defer addResp.Body.Close()

	// assert
	if registerResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected register status: got %d, want %d", registerResp.StatusCode, http.StatusOK)
	}

	if deleteResp.StatusCode != http.StatusOK {
		t.Errorf("unexpected delete status: got %d, want %d", deleteResp.StatusCode, http.StatusOK)
	}

	if addResp.StatusCode == http.StatusOK {
		t.Errorf("unexpected add status for deleted chat: got %d, want non-200", addResp.StatusCode)
	}
}

func TestScrapperServer_DeleteNonExistingChat(t *testing.T) {
	// arrange
	ts := newTestScrapperServer()

	deleteReq, err := http.NewRequest(http.MethodDelete, ts.URL+"/tg-chat/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	// act
	deleteResp, err := http.DefaultClient.Do(deleteReq)
	if err != nil {
		t.Fatal(err)
	}
	defer deleteResp.Body.Close()

	// assert
	if deleteResp.StatusCode != http.StatusNotFound {
		t.Errorf("unexpected delete status: got %d, want %d", deleteResp.StatusCode, http.StatusNotFound)
	}
}

func newTestScrapperServer() *httptest.Server {
	chatRepository := memory.NewChatRepository()
	subscriptionRepository := memory.NewSubscriptionRepository()
	subscriptionService := service.NewSubscriptionService(chatRepository, subscriptionRepository)

	mux := http.NewServeMux()
	mux.Handle("/tg-chat/", NewTgChatHandler(subscriptionService))
	mux.Handle("/links", NewLinksHandler(subscriptionService))

	return httptest.NewServer(mux)
}
