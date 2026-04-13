package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
)

type Job interface {
	Run(ctx context.Context) error
}

// Starts job with time interval
type Scheduler struct {
	scheduler gocron.Scheduler
	job       Job
	interval  time.Duration
	logger    *slog.Logger
}

func New(job Job, interval time.Duration, logger *slog.Logger) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		scheduler: s,
		job:       job,
		interval:  interval,
		logger:    logger,
	}, nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	_, err := s.scheduler.NewJob(
		gocron.DurationJob(s.interval),
		gocron.NewTask(func() {
			if err := s.job.Run(ctx); err != nil {
				s.logger.Error("scheduled job failed", "error", err)
				return
			}
		}),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return err
	}

	s.scheduler.Start()
	s.logger.Info("scheduler started")

	return nil
}

func (s *Scheduler) Stop() error {
	return s.scheduler.Shutdown()
}
