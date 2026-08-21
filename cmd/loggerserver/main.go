package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	pb "task_1_hamrah/api"
	"task_1_hamrah/internal/config"
	"task_1_hamrah/internal/repository"
	"task_1_hamrah/internal/service"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

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

	loggedEventRepo := repository.NewPostgresLoggedEventRepository(db)
	if err := loggedEventRepo.Migrate(context.Background()); err != nil {
		fmt.Println("Failed to migrate logged_events table: ", err)
		return
	}

	loggerService := service.NewLoggerService(loggedEventRepo)

	listen, err := net.Listen("tcp", ":"+cfg.LoggerPort)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLoggerServer(grpcServer, loggerService)

	fmt.Println("Logger service listening on :" + cfg.LoggerPort)
	if err := grpcServer.Serve(listen); err != nil {
		fmt.Println("Error serving:", err)
	}
}
