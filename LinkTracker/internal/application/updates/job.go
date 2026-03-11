package updates

import (
	"context"
	"log/slog"
)

type Job struct {
	logger  *slog.Logger
	checker *Checker
}

func NewJob(logger *slog.Logger, checker *Checker) Job {
	return Job{
		logger:  logger,
		checker: checker,
	}
}

func (j Job) Run(ctx context.Context) error {
	j.logger.Info("checking tracked links for updates")

	if err := j.checker.Check(ctx); err != nil {
		return err
	}

	j.logger.Info("tracked links updates check finished")

	return nil
}