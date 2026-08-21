package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	pb "task_1_hamrah/api"
	"task_1_hamrah/internal/config"
	eventlogger "task_1_hamrah/internal/event_logger"
	"task_1_hamrah/internal/repository"
	"task_1_hamrah/internal/service"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedCalculatorServer
	calculator pb.CalculatorServer
}

func (s *server) Calc(ctx context.Context, req *pb.CalcRequest) (*pb.CalcAnswer, error) {
	return s.calculator.Calc(ctx, req)
}

func (s *server) Register(ctx context.Context, req *pb.AuthRequest) (*pb.AuthReply, error) {
	return s.calculator.Register(ctx, req)
}

func (s *server) Login(ctx context.Context, req *pb.AuthRequest) (*pb.AuthReply, error) {
	return s.calculator.Login(ctx, req)
}

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DBDSN())
	if err != nil {
		fmt.Println("Failed to connect database: ", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Println("Failed to ping database: ", err)
		return
	}
	fmt.Println("Successfully connected to database")

	ctx := context.Background()

	userRepo := repository.NewPostgresUserRepository(db)
	calcHistoryRepo := repository.NewPostgresCalcHistoryRepository(db)

	migrations := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"users", userRepo.Migrate},
		{"calc_history", calcHistoryRepo.Migrate},
	}
	for _, m := range migrations {
		if err := m.fn(ctx); err != nil {
			fmt.Printf("Failed to migrate %s: %v\n", m.name, err)
			return
		}
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Println("Failed to connect to redis: ", err)
		return
	}
	fmt.Println("Successfully connected to redis")
	defer redisClient.Close()

	throttle := repository.NewRedisLoginThrottler(redisClient)

	eventLogger, err := eventlogger.NewGRPCEventLogger(cfg.LoggerAddr)
	if err != nil {
		fmt.Println("Failed to connect to logger calculator: ", err)
		return
	}
	defer eventLogger.Close()

	calculatorService := service.NewCalculatorService(userRepo, calcHistoryRepo, eventLogger, throttle)

	listen, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}

	myServer := grpc.NewServer()
	pb.RegisterCalculatorServer(myServer, &server{calculator: calculatorService})

	fmt.Println("Listening on :" + cfg.GRPCPort)
	if err := myServer.Serve(listen); err != nil {
		fmt.Println("Error serving:", err)
	}
}
