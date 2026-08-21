package service

import (
	"io"
	"log"
	pb "task_1_hamrah/api"
	"task_1_hamrah/internal/repository"
)

type loggerService struct {
	pb.UnimplementedLoggerServer
	events repository.LoggedEventRepository
}

func NewLoggerService(events repository.LoggedEventRepository) pb.LoggerServer {
	return &loggerService{events: events}
}

func (s *loggerService) StreamEvents(stream pb.Logger_StreamEventsServer) error {
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.EventReply{Received: true})
		}
		if err != nil {
			return err
		}
		if err := s.events.Insert(stream.Context(), event.GetUser(), event.GetTimestamp().AsTime()); err != nil {
			log.Printf("failed to insert logged event: %v", err)
		}
	}
}
