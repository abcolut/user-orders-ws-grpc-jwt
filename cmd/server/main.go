package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"database/sql"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"my_api/internal/auth"
	dbpkg "my_api/internal/db"
	"my_api/internal/handlers"
	"my_api/internal/ws"
	pb "my_api/proto"
)

// ---------- gRPC service ----------

type orderService struct {
	pb.UnimplementedOrderServiceServer

	repo *dbpkg.OrderRepo
	hub  *ws.Hub // TODO: использовать (например, уведомлять WS о новых заказах)
}

func (s *orderService) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	userID := int(req.GetUserId())
	if userID == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id обязателен")
	}

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

	return &pb.ListOrdersResponse{Orders: pbOrders}, nil
}

// ---------- main ----------

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("завершение с ошибкой", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Warn("не удалось загрузить .env", "err", err)
	}

	// 2. Конфиг (все обязательные переменные проверяются сразу).
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// 3. БД.
	db, err := dbpkg.InitDB(cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Error("db.Close", "err", cerr)
		}
	}()

	// 4. Hub и репозиторий.
	hub := ws.NewHub()
	handlers.SetHub(hub)

	orderRepo := dbpkg.NewOrderRepo(db, hub)
	if orderRepo == nil {
		return errors.New("не удалось создать OrderRepo")
	}

	// 5. gRPC сервер.
	grpcLis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, &orderService{repo: orderRepo, hub: hub})

	grpcErrCh := make(chan error, 1)
	go func() {
		logger.Info("gRPC сервер запущен", "addr", grpcLis.Addr().String())
		if err := grpcServer.Serve(grpcLis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			grpcErrCh <- err
		}
	}()

	// 6. HTTP сервер.
	mux := buildMux(db, orderRepo, cfg.Secret)

	httpSrv := &http.Server{
		Addr:              ":" + cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	httpErrCh := make(chan error, 1)
	go func() {
		logger.Info("HTTP сервер запущен", "addr", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			httpErrCh <- err
		}
	}()

	// 7. Ждём сигнал или падение любого из серверов.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logger.Info("получен сигнал, останавливаемся")
	case err := <-grpcErrCh:
		logger.Error("gRPC упал", "err", err)
	case err := <-httpErrCh:
		logger.Error("HTTP упал", "err", err)
	}

	// 8. Graceful shutdown с общим дедлайном.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var shutdownErr error

	// HTTP — мягко.
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		shutdownErr = errors.Join(shutdownErr, err)
		logger.Error("HTTP Shutdown", "err", err)
	}

	// gRPC — мягко; если не успели — жёстко.
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-shutdownCtx.Done():
		logger.Warn("gRPC GracefulStop не успел, вызываем Stop")
		grpcServer.Stop()
		<-done
	}

	logger.Info("остановлено")
	return shutdownErr
}

// ---------- конфиг ----------

type config struct {
	DBPath   string
	GRPCPort string
	HTTPAddr string
	Secret   []byte
}

func loadConfig() (config, error) {
	cfg := config{
		DBPath:   getEnvDefault("DB_PATH", "my_api.db"),
		GRPCPort: getEnvDefault("GRPC_PORT", "50051"),
		HTTPAddr: getEnvDefault("HTTP_ADDR", "8080"),
	}

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		return cfg, errors.New("SECRET_KEY не задана")
	}
	if len(secretKey) < 32 {
		return cfg, errors.New("SECRET_KEY должен быть не короче 32 байт")
	}
	cfg.Secret = []byte(secretKey)

	return cfg, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ---------- маршруты ----------

func buildMux(db *sql.DB, orderRepo *dbpkg.OrderRepo, secret []byte) http.Handler {
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

	return mux
}
