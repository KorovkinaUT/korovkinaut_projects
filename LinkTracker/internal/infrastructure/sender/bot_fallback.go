package sender

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	appsender "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/sender"
)

// Uses primary reciever, but if it fails, uses fallback reciever
type BotFallbackSender struct {
	primary  appsender.MessageSender
	fallback appsender.MessageSender
	logger   *slog.Logger
}

var _ appsender.MessageSender = (*BotFallbackSender)(nil)

func NewBotFallbackSender(
	primary appsender.MessageSender,
	fallback appsender.MessageSender,
	logger *slog.Logger,
) *BotFallbackSender {
	return &BotFallbackSender{
		primary:  primary,
		fallback: fallback,
		logger:   logger,
	}
}

func (s *BotFallbackSender) SendUpdate(ctx context.Context, msg appsender.UpdateMessage) error {
	err := s.primary.SendUpdate(ctx, msg)
	if err == nil {
		return nil
	}

	s.warn("primary update sender failed, using fallback sender", "error", err)

	fallbackErr := s.fallback.SendUpdate(ctx, msg)
	if fallbackErr != nil {
		return errors.Join(
			fmt.Errorf("primary sender send update failed: %w", err),
			fmt.Errorf("fallback sender send update failed: %w", fallbackErr),
		)
	}

	return nil
}

func (s *BotFallbackSender) SendProblems(ctx context.Context, msg appsender.ProblemsMessage) error {
	err := s.primary.SendProblems(ctx, msg)
	if err == nil {
		return nil
	}

	s.warn("primary update sender failed, using fallback sender", "error", err)

	fallbackErr := s.fallback.SendProblems(ctx, msg)
	if fallbackErr != nil {
		return errors.Join(
			fmt.Errorf("primary sender send problems failed: %w", err),
			fmt.Errorf("fallback sender send problems failed: %w", fallbackErr),
		)
	}

	return nil
}

func (s *BotFallbackSender) Close() error {
	var result error

	if closer, ok := s.primary.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("close primary sender: %w", err))
		}
	}

	if closer, ok := s.fallback.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("close fallback sender: %w", err))
		}
	}

	return result
}

func (s *BotFallbackSender) warn(msg string, args ...any) {
	if s.logger == nil {
		return
	}

	s.logger.Warn(msg, args...)
}
