package event_logger

import (
	"context"
	"io"
	"task_1_hamrah/api"
)

type EventLogger interface {
	LogEvent(ctx context.Context, event *api.Event) error
	io.Closer
}
