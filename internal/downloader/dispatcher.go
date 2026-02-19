package downloader

import (
	"context"
	"errors"
	"log/slog"

	"github.com/cesargomez89/navidrums/internal/domain"
)

var ErrUnknownJobType = errors.New("unknown job type")

type JobHandler interface {
	Handle(ctx context.Context, job *domain.Job, logger *slog.Logger) error
}

type Dispatcher struct {
	handlers map[domain.JobType]JobHandler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[domain.JobType]JobHandler),
	}
}

func (d *Dispatcher) Register(jobType domain.JobType, handler JobHandler) {
	d.handlers[jobType] = handler
}

func (d *Dispatcher) Dispatch(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	handler, ok := d.handlers[job.Type]
	if !ok {
		return ErrUnknownJobType
	}
	return handler.Handle(ctx, job, logger)
}
