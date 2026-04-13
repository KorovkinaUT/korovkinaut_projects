package bothttp

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Handler for /updates
type UpdatesHandler struct {
	sendMessage func(chatID int64, text string) error
}

func NewUpdatesHandler(sendMessage func(chatID int64, text string) error) *UpdatesHandler {
	return &UpdatesHandler{
		sendMessage: sendMessage,
	}
}

func (h *UpdatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req LinkUpdate
	if err := decoder.Decode(&req); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"BadRequest",
			"invalid request body",
			err.Error(),
		)
		return
	}

	// Check that request has all fields
	if !hasRequiredFields(req) {
		writeError(
			w,
			http.StatusBadRequest,
			"BadRequest",
			"missing required fields",
			"all fields must be present",
		)
		return
	}

	// Sends notification to chats
	for _, chatID := range req.TgChatIDs {
		if err := h.sendMessage(chatID, req.Description); err != nil {
			writeError(
				w,
				http.StatusInternalServerError,
				"InternalServerError",
				"failed to send telegram message",
				err.Error(),
			)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func hasRequiredFields(req LinkUpdate) bool {
	if req.ID == 0 {
		return false
	}

	if req.URL == "" {
		return false
	}

	if req.Description == "" {
		return false
	}

	if req.TgChatIDs == nil {
		return false
	}

	return true
}

func writeError(
	w http.ResponseWriter,
	status int,
	exceptionName string,
	description string,
	exceptionMessage string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := ApiErrorResponse{
		Description:      description,
		Code:             strconv.Itoa(status),
		ExceptionName:    exceptionName,
		ExceptionMessage: exceptionMessage,
	}

	_ = json.NewEncoder(w).Encode(resp)
}
