package receiver

import (
	"context"
	"errors"
	"log/slog"
)

// For fallback cases, recieves from different places
type BotCompositeReceiver struct {
	receivers []MessageReceiver
}

var _ MessageReceiver = (*BotCompositeReceiver)(nil)

func NewCompositeReceiver(receivers ...MessageReceiver) *BotCompositeReceiver {
	return &BotCompositeReceiver{
		receivers: receivers,
	}
}

func (r *BotCompositeReceiver) Start(ctx context.Context, logger *slog.Logger, stop context.CancelFunc) {
	for _, receiver := range r.receivers {
		receiver.Start(ctx, logger, stop)
	}
}

func (r *BotCompositeReceiver) Shutdown(ctx context.Context) error {
	var result error

	for _, receiver := range r.receivers {
		if err := receiver.Shutdown(ctx); err != nil {
			result = errors.Join(result, err)
		}
	}

	return result
}
