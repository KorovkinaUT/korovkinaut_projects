package scrapperhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/repository"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
)

// Handler for /tg-chat/{id}
type TgChatHandler struct {
	subscriptions *service.SubscriptionService
}

func NewTgChatHandler(subscriptions *service.SubscriptionService) *TgChatHandler {
	return &TgChatHandler{
		subscriptions: subscriptions,
	}
}

func (h *TgChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	chatID, ok := parseChatIDFromPath(r.URL.Path)
	if !ok || chatID <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid request parameters", "invalid chat id")
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.handleRegisterChat(ctx, w, chatID)
	case http.MethodDelete:
		h.handleDeleteChat(ctx, w, chatID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *TgChatHandler) handleRegisterChat(ctx context.Context, w http.ResponseWriter, chatID int64) {
	err := h.subscriptions.RegisterChat(ctx, chatID)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch err {
	case repository.ErrChatAlreadyExists:
		writeAPIError(w, http.StatusConflict, "chat already exists", err.Error())
	default:
		writeAPIError(w, http.StatusBadRequest, "failed to register chat", err.Error())
	}
}

func (h *TgChatHandler) handleDeleteChat(ctx context.Context, w http.ResponseWriter, chatID int64) {
	err := h.subscriptions.DeleteChat(ctx, chatID)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch err {
	case repository.ErrChatNotFound:
		writeAPIError(w, http.StatusNotFound, "chat not found", err.Error())
	default:
		writeAPIError(w, http.StatusBadRequest, "failed to delete chat", err.Error())
	}
}

func parseChatIDFromPath(path string) (int64, bool) {
	const prefix = "/tg-chat/"
	if !strings.HasPrefix(path, prefix) {
		return 0, false
	}

	idPart := strings.TrimPrefix(path, prefix)

	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}

func writeAPIError(w http.ResponseWriter, status int, description string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := ApiErrorResponse{
		Description:      description,
		Code:             strconv.Itoa(status),
		ExceptionName:    http.StatusText(status),
		ExceptionMessage: message,
	}

	_ = json.NewEncoder(w).Encode(resp)
}
