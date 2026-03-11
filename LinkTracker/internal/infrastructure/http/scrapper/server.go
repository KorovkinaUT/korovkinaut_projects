package scrapperhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/service"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(addr string, subscriptions *service.SubscriptionService) *Server {
	mux := http.NewServeMux()

	mux.Handle("/tg-chat/", NewTgChatHandler(subscriptions))
	mux.Handle("/links", NewLinksHandler(subscriptions))

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

func (s *Server) Start(logger *slog.Logger, stop func()) {
	go func() {
		logger.Info("scrapper http server started", "addr", s.httpServer.Addr)

		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("scrapper http server failed", "error", err)
			stop()
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}