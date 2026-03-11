package scrapperhttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
)

// Handler for /links
type LinksHandler struct {
	subscriptions *service.SubscriptionService
}

func NewLinksHandler(subscriptions *service.SubscriptionService) *LinksHandler {
	return &LinksHandler{
		subscriptions: subscriptions,
	}
}

func (h *LinksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	chatID, ok := parseChatIDHeader(r.Header)
	if !ok || chatID <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid request parameters", "invalid Tg-Chat-Id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleListLinks(w, chatID)
	case http.MethodPost:
		h.handleAddLink(w, r, chatID)
	case http.MethodDelete:
		h.handleRemoveLink(w, r, chatID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *LinksHandler) handleListLinks(w http.ResponseWriter, chatID int64) {
	links, err := h.subscriptions.ListLinks(chatID)
	if err != nil {
		switch err {
		case repository.ErrChatNotFound:
			writeAPIError(w, http.StatusNotFound, "chat not found", err.Error())
		default:
			writeAPIError(w, http.StatusBadRequest, "failed to list links", err.Error())
		}
		return
	}

	respLinks := make([]LinkResponse, 0, len(links))
	for _, link := range links {
		respLinks = append(respLinks, LinkResponse{
			ID:   link.ID,
			URL:  link.URL,
			Tags: link.Tags,
		})
	}

	resp := ListLinksResponse{
		Links: respLinks,
		Size:  int32(len(respLinks)),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *LinksHandler) handleAddLink(w http.ResponseWriter, r *http.Request, chatID int64) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req AddLinkRequest
	if err := decoder.Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Link == "" || req.Tags == nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request parameters", "link and tags fields must be present")
		return
	}

	link, err := h.subscriptions.AddLink(chatID, req.Link, req.Tags)
	if err != nil {
		switch err {
		case repository.ErrChatNotFound:
			writeAPIError(w, http.StatusNotFound, "chat not found", err.Error())
		case repository.ErrLinkAlreadyTracked:
			writeAPIError(w, http.StatusConflict, "link already tracked", err.Error())
		default:
			writeAPIError(w, http.StatusBadRequest, "failed to add link", err.Error())
		}
		return
	}

	resp := LinkResponse{
		ID:   link.ID,
		URL:  link.URL,
		Tags: link.Tags,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *LinksHandler) handleRemoveLink(w http.ResponseWriter, r *http.Request, chatID int64) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req RemoveLinkRequest
	if err := decoder.Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Link == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid request parameters", "link field must be present")
		return
	}

	link, err := h.subscriptions.RemoveLink(chatID, req.Link)
	if err != nil {
		switch err {
		case repository.ErrChatNotFound, repository.ErrLinkNotFound:
			writeAPIError(w, http.StatusNotFound, "chat or link not found", err.Error())
		default:
			writeAPIError(w, http.StatusBadRequest, "failed to remove link", err.Error())
		}
		return
	}

	resp := LinkResponse{
		ID:   link.ID,
		URL:  link.URL,
		Tags: link.Tags,
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseChatIDHeader(header http.Header) (int64, bool) {
	value := header.Get("Tg-Chat-Id")
	if value == "" {
		return 0, false
	}

	chatID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}

	return chatID, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
