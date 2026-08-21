package event_logger

import (
	"context"
	"errors"
	"log"
	"sync"
	"task_1_hamrah/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcEventLogger struct {
	conn   *grpc.ClientConn
	events chan *api.Event
	stream api.Logger_StreamEventsClient
	wg     sync.WaitGroup
}

func NewGRPCEventLogger(addr string) (EventLogger, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	stream, err := api.NewLoggerClient(conn).StreamEvents(context.Background())
	if err != nil {
		conn.Close()
		return nil, err
	}

	l := &grpcEventLogger{
		conn:   conn,
		stream: stream,
		events: make(chan *api.Event, 200),
	}

	l.wg.Add(1)
	go l.run()

	return l, nil
}

func (l *grpcEventLogger) LogEvent(ctx context.Context, event *api.Event) error {
	select {
	case l.events <- event:
		return nil
	default:
		return errors.New("event buffer full, dropping event")
	}
}

func (l *grpcEventLogger) run() {
	defer l.wg.Done()
	for event := range l.events {
		if err := l.stream.Send(event); err != nil {
			log.Printf("failed to send event: %v", err)
		}
	}
}

func (l *grpcEventLogger) Close() error {
	close(l.events)
	l.wg.Wait()

	_, err := l.stream.CloseAndRecv()
	closeErr := l.conn.Close()
	if err != nil {
		return err
	}
	return closeErr
}
