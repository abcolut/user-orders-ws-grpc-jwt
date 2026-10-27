package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "my_api/proto"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Не удалось подключиться к gRPC: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.ListOrdersRequest{
		UserId: 1, // <-- вот он, user_id=1
	}

	resp, err := client.ListOrders(ctx, req)
	if err != nil {
		log.Printf("Ошибка gRPC: %v", err)
		return
	}

	log.Println("Успешный ответ от gRPC:")
	log.Printf("%+v", resp)
}
