package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "task_1_hamrah/api"
)

func main() {
	connection, err := grpc.NewClient("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	defer connection.Close()
	client := pb.NewCalculatorClient(connection)

	username := authenticate(client)
	var accum float64

	unaryOps := map[string]bool{"sqrt": true, "ln": true, "exp": true, "sin": true, "cos": true, "tan": true}

	fmt.Print(0)

	for {
		var operation string
		_, _ = fmt.Fscan(os.Stdin, &operation)
		if operation == "exit" {
			break
		}

		var num2 float64
		if !unaryOps[operation] {
			_, _ = fmt.Fscan(os.Stdin, &num2)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req := &pb.CalcRequest{Name: username, Num1: accum, Num2: num2, Operation: operation}
		reply, err := client.Calc(ctx, req)
		cancel()
		if err != nil {
			fmt.Println("Error:", status.Convert(err).Message())
		}

		accum = reply.GetResult()
		fmt.Print(accum)
	}
}

func authenticate(client pb.CalculatorClient) string {
	for {
		var choice int
		var username string
		var password string
		fmt.Println("1) Register\n2) Login")
		_, _ = fmt.Fscanf(os.Stdin, "%d %s %s", &choice, &username, &password)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req := &pb.AuthRequest{Username: username, Password: password}

		var reply *pb.AuthReply
		var err error
		if choice == 1 {
			reply, err = client.Register(ctx, req)
		} else {
			reply, err = client.Login(ctx, req)
		}
		cancel()

		if err != nil {
			fmt.Println("Error:", status.Convert(err).Message())
			continue
		}

		fmt.Println(reply.GetMessage())
		return username
	}
}
