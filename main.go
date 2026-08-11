package main

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net"
	"time"

	pb "task_1_hamrah/api"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedCalculatorServer
	database *sql.DB
	redis    *redis.Client
}

func (s *server) Calc(ctx context.Context, req *pb.CalcRequest) (*pb.CalcAnswer, error) {
	query := "INSERT INTO requests_log (client_name) VALUES ($1)"

	_, err := s.database.Exec(query, req.GetName())
	if err != nil {
		fmt.Println("Error inserting data")
		return nil, err
	}

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

	query = "INSERT INTO calc_history (num1, num2, operation, answer, name) VALUES ($1, $2, $3, $4, $5)"
	_, err = s.database.Exec(query, req.GetNum1(), req.GetNum2(), req.GetOperation(), answer, req.GetName())
	if err != nil {
		fmt.Println("Error inserting data")
		return nil, err
	}

	return &pb.CalcAnswer{Result: answer}, nil
}

func (s *server) Register(ctx context.Context, req *pb.AuthRequest) (*pb.AuthReply, error) {
	var exists bool
	err := s.database.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE username = $1)", req.GetUsername()).Scan(&exists)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	if exists {
		return nil, status.Errorf(codes.AlreadyExists, "username already taken")
	}

	_, err = s.database.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)

	}

	return &pb.AuthReply{Message: "registered successfully"}, nil
}

func (s *server) Login(ctx context.Context, req *pb.AuthRequest) (*pb.AuthReply, error) {
	key := "login_fail:" + req.GetUsername()
	penaltyKey := "penalty:" + req.GetUsername()

	count, err := s.redis.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return nil, status.Errorf(codes.Internal, "redis error: %v", err)
	}
	if count >= 5 {
		ttl, err := s.redis.TTL(ctx, key).Result()
		s.recordFailedLogin(ctx, key, penaltyKey)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "redis error: %v", err)
		}
		return nil, status.Errorf(codes.ResourceExhausted,
			"too many failed login attempts, try again in %d minutes", int(ttl.Minutes()))
	}

	var password string
	err = s.database.QueryRow("SELECT password FROM users WHERE username = $1", req.GetUsername()).Scan(&password)
	if err == sql.ErrNoRows {
		s.recordFailedLogin(ctx, key, penaltyKey)
		return nil, status.Errorf(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	if password != req.GetPassword() {
		s.recordFailedLogin(ctx, key, penaltyKey)
		return nil, status.Errorf(codes.Unauthenticated, "wrong password")
	}

	s.redis.Del(ctx, key)
	return &pb.AuthReply{Message: "logged in successfully"}, nil
}

func (s *server) recordFailedLogin(ctx context.Context, key string, penaltyKey string) {
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		fmt.Println("Error incrementing:", err)
		return
	}

	if count < 5 {
		if count == 1 {
			s.redis.Expire(ctx, key, time.Minute)
		}
		return
	}

	penalty, err := s.redis.Get(ctx, penaltyKey).Int()
	if err != nil && err != redis.Nil {
		fmt.Println("Error reading penalty:", err)
		return
	}
	if penalty == 0 {
		penalty = 1
	} else {
		penalty *= 2
	}

	s.redis.Set(ctx, penaltyKey, penalty, 0)
	s.redis.Expire(ctx, key, time.Duration(penalty)*time.Minute)
}

func main() {
	sqlConnectionStr := "host=postgres-db user=postgres password=radan1386 dbname=task_1_hamrah sslmode=disable"
	db, err := sql.Open("postgres", sqlConnectionStr)
	if err != nil {
		fmt.Println("Failed to connect database: ", err)
		return
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Println("Failed to ping database: ", err)
		return
	}
	fmt.Println("Successfully connected to database")

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (username TEXT PRIMARY KEY, password TEXT NOT NULL)")
	if err != nil {
		fmt.Println("Failed to create users table: ", err)
		return
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS requests_log (id SERIAL PRIMARY KEY, client_name TEXT)")
	if err != nil {
		fmt.Println("Failed to create requests_log table: ", err)
		return
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS calc_history (id SERIAL PRIMARY KEY, num1 DOUBLE PRECISION, num2 DOUBLE PRECISION, operation TEXT, answer DOUBLE PRECISION, name TEXT)")
	if err != nil {
		fmt.Println("Failed to create calc_history table: ", err)
		return
	}

	_, err = db.Exec("ALTER TABLE calc_history ADD COLUMN IF NOT EXISTS name TEXT")
	if err != nil {
		fmt.Println("Failed to alter calc_history table: ", err)
		return
	}

	_, err = db.Exec("ALTER TABLE calc_history ALTER COLUMN num1 TYPE DOUBLE PRECISION, ALTER COLUMN num2 TYPE DOUBLE PRECISION, ALTER COLUMN answer TYPE DOUBLE PRECISION")
	if err != nil {
		fmt.Println("Failed to alter calc_history column types: ", err)
		return
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	err = redisClient.Ping(context.Background()).Err()
	if err != nil {
		fmt.Println("Failed to connect to redis: ", err)
		return
	}
	fmt.Println("Successfully connected to redis")

	defer redisClient.Close()

	listen, error := net.Listen("tcp", ":8080")
	if error != nil {
		fmt.Println("Error listening:", error)
		return
	}

	myServer := grpc.NewServer()
	pb.RegisterCalculatorServer(myServer, &server{database: db, redis: redisClient})

	fmt.Println("Listening on :8080")
	err = myServer.Serve(listen)
	if err != nil {
		fmt.Println("Error serving:", err)
	}

}
