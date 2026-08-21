package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"math"
	"task_1_hamrah/api"
	pb "task_1_hamrah/api"
	"task_1_hamrah/internal/event_logger"
	"task_1_hamrah/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type calculatorService struct {
	users       repository.UserRepository
	calcHistory repository.CalcHistoryRepository
	throttle    repository.LoginThrottler
	eventLogger event_logger.EventLogger
	pb.UnimplementedCalculatorServer
}

func NewCalculatorService(
	users repository.UserRepository,
	calcHistory repository.CalcHistoryRepository,
	eventLogger event_logger.EventLogger,
	throttle repository.LoginThrottler,
) pb.CalculatorServer {
	return &calculatorService{
		users:       users,
		calcHistory: calcHistory,
		eventLogger: eventLogger,
		throttle:    throttle,
	}
}

func (s *calculatorService) Calc(ctx context.Context, req *pb.CalcRequest) (*pb.CalcAnswer, error) {

	var answer float64
	switch req.GetOperation() {
	case "add":
		answer = req.GetNum1() + req.GetNum2()
	case "sub":
		answer = req.GetNum1() - req.GetNum2()
	case "mul":
		answer = req.GetNum1() * req.GetNum2()
	case "div":
		answer = req.GetNum1() / req.GetNum2()
	case "mod":
		answer = math.Mod(req.GetNum1(), req.GetNum2())
	case "pow":
		answer = math.Pow(req.GetNum1(), req.GetNum2())
	case "sqrt":
		answer = math.Sqrt(req.GetNum1())
	case "ln":
		answer = math.Log(req.GetNum1())
	case "exp":
		answer = math.Exp(req.GetNum1())
	case "sin":
		answer = math.Sin(req.GetNum1())
	case "cos":
		answer = math.Cos(req.GetNum1())
	case "tan":
		answer = math.Tan(req.GetNum1())
	case "del":
		answer = 0
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Invalid operation")
	}

	event := &api.Event{
		User:      req.GetName(),
		Timestamp: timestamppb.Now(),
	}
	if err := s.eventLogger.LogEvent(ctx, event); err != nil {
		log.Printf("failed to log event: %v", err)
	}

	return &pb.CalcAnswer{Result: answer}, nil
}

func (s *calculatorService) Register(ctx context.Context, req *pb.AuthRequest) (*pb.AuthReply, error) {
	exists, err := s.users.Exists(ctx, req.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	if exists {
		return nil, status.Errorf(codes.AlreadyExists, "username already taken")
	}

	if err := s.users.Create(ctx, req.GetUsername(), req.GetPassword()); err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	return &pb.AuthReply{Message: "registered successfully"}, nil
}

func (s *calculatorService) Login(ctx context.Context, req *pb.AuthRequest) (*pb.AuthReply, error) {
	username := req.GetUsername()

	count, err := s.throttle.FailCount(ctx, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "redis error: %v", err)
	}
	if count >= 5 {
		ttl, err := s.throttle.PenaltyTTL(ctx, username)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "redis error: %v", err)
		}
		if err := s.throttle.RecordFailure(ctx, username); err != nil {
			return nil, status.Errorf(codes.Internal, "redis error: %v", err)
		}
		return nil, status.Errorf(codes.ResourceExhausted,
			"too many failed login attempts, try again in %d minutes", int(ttl.Minutes()))
	}

	password, err := s.users.GetPassword(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		if rerr := s.throttle.RecordFailure(ctx, username); rerr != nil {
			return nil, status.Errorf(codes.Internal, "redis error: %v", rerr)
		}
		return nil, status.Errorf(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	if password != req.GetPassword() {
		if rerr := s.throttle.RecordFailure(ctx, username); rerr != nil {
			return nil, status.Errorf(codes.Internal, "redis error: %v", rerr)
		}
		return nil, status.Errorf(codes.Unauthenticated, "wrong password")
	}

	if err := s.throttle.Reset(ctx, username); err != nil {
		return nil, status.Errorf(codes.Internal, "redis error: %v", err)
	}
	return &pb.AuthReply{Message: "logged in successfully"}, nil
}
