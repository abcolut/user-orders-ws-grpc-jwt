package main

import (
	// "encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"my_api/internal/auth"
	dbpkg "my_api/internal/db"
	"my_api/internal/handlers"
	"my_api/internal/ws"
	pb "my_api/proto"

	"context"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type orderService_old struct {
	pb.UnimplementedOrderServiceServer

	repo *dbpkg.OrderRepo
	hub  *ws.Hub
}

func (s *orderService) ListOrders_old(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	// TODO userID := req.UserId
	userID := 1

	orders, err := s.repo.ListOrdersByUserId(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "не удалось получить заказы: %v", err)
	}

	pbOrders := make([]*pb.Order, 0, len(orders))
	for _, o := range orders {
		pbOrders = append(pbOrders, &pb.Order{
			Id:        int64(o.ID),
			UserId:    int32(o.UserID),
			Status:    o.Status,
			Item:      o.Item,
			Quantity:  int32(o.Quantity),
			Customer:  o.Customer,
			CreatedAt: o.CreatedAt,
		})
	}

	return &pb.ListOrdersResponse{
		Orders: pbOrders,
	}, nil
}

func main_old() {
	// 1. Инициализация БД
	dir, _ := os.Getwd()
	log.Printf("Рабочая директория: %s", dir)

	dbPath := "my_api.db"
	db, err := dbpkg.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Не удалось инициализировать БД: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("db.Close: %v", err)
		}
	}()

	// 2. Загрузка .env
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Не удалось загрузить .env: %v", err)
	}

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		log.Fatal("Переменная SECRET_KEY не задана")
	}
	if len(secretKey) < 32 {
		log.Fatal("SECRET_KEY должен быть не короче 32 байт")
	}
	secret := []byte(secretKey)

	// 3. WebSocket Hub
	hub := ws.NewHub()
	handlers.SetHub(hub)

	// 4. Репозиторий (один экземпляр)
	orderRepo := dbpkg.NewOrderRepo(db, hub)
	if orderRepo == nil {
		log.Fatal("Не удалось создать OrderRepo")
	}

	// 5. gRPC сервер + сервис с зависимостями
	grpc_port := os.Getenv("GRPC_PORT")
	if grpc_port == "" {
		log.Fatal("env.GRPC_PORT не задана")
	}

	lis, err := net.Listen("tcp", ":"+grpc_port)
	if err != nil {
		log.Fatalf("Не удалось слушать порт %s: %v", grpc_port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, &orderService{
		repo: orderRepo,
		hub:  hub,
	})

	go func() {
		log.Println("gRPC сервер запущен на", os.Getenv("GRPC_PORT"))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC Serve завершился с ошибкой: %v", err)
		}
	}()

	// 6. HTTP сервер (mux)
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", handlers.AuthHtmlHandler)
	mux.HandleFunc("/api/auth", handlers.AuthApiHandler(db, secret))

	mux.HandleFunc("/reg", handlers.RegHtmlHandler)
	mux.HandleFunc("/api/reg", handlers.RegApiHandler(db))

	mux.HandleFunc("/orders", handlers.OrdersHtmlHandler)

	mux.Handle("/api/orders", auth.AuthMiddleware(secret, http.HandlerFunc(handlers.OrdersApiHandler(orderRepo))))

	mux.HandleFunc("/ws_html", handlers.WsHtmlHandler)
	mux.HandleFunc("/api/ws", handlers.WsApiHandler)

	mux.Handle("/secret", auth.AuthMiddleware(secret, http.HandlerFunc(handlers.SecretHandler)))
	mux.HandleFunc("/hello", handlers.HelloHandler)
	mux.HandleFunc("/", handlers.IndexHandler)

	http_addr := os.Getenv("HTTP_ADDR")

	if http_addr == "" {
		log.Fatal("env.HTTP_ADDR не задана")
	}

	log.Printf("HTTP сервер запущен: http://localhost:%s", http_addr)
	if err := http.ListenAndServe(":"+http_addr, mux); err != nil {
		log.Fatalf("Ошибка HTTP сервера: %v", err)
	}
}
