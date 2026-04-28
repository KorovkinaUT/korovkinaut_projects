package bothttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(addr string, sendMessage func(chatID int64, text string) error) *Server {
	mux := http.NewServeMux()
	mux.Handle("/updates", NewUpdatesHandler(sendMessage))

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &Server{
		httpServer: srv,
	}
}

func (s *Server) Start(logger *slog.Logger) error {
	logger.Info("bot http server started", "addr", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve bot http server: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
