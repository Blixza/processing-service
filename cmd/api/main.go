// @title Image Processor API
// @version 1.0
// @description This is a distributed image processing service.

// @contact.name Blixza
// @contact.url https://github.com/Blixza

// @host localhost:8081
// @BasePath /
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"main/config"
	_ "main/docs"
	"main/internal/database"
	_ "main/internal/domain/request"
	"main/internal/logger"
	"main/internal/transport/grpc/worker"
	"main/internal/transport/rabbitmq"
	"main/server"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := godotenv.Load()
	if err != nil {
		log.Printf("Failed to load env: %v\n", err)
		return
	}

	l, rabbit, infra, workerClient, serverCfg := initApp(ctx)
	defer rabbit.Conn.Close()

	mux := setupRoutes(l, rabbit, infra, workerClient)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", serverCfg.Port),
		Handler:           mux,
		ReadTimeout:       time.Duration(serverCfg.ReadTimeoutSec) * time.Second,
		WriteTimeout:      time.Duration(serverCfg.WriteTimeoutSec) * time.Second,
		IdleTimeout:       time.Duration(serverCfg.IdleTimeoutSec) * time.Second,
		ReadHeaderTimeout: time.Duration(serverCfg.ReadHeaderTimeoutSec) * time.Second,
	}

	go func() {
		l.Info("HTTP server starting", zap.String("addr", httpServer.Addr))

		err = httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	l.Info("Gracefully shutting down...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(serverCfg.ReadHeaderTimeoutSec)*time.Second,
	)
	defer cancel()

	err = httpServer.Shutdown(shutdownCtx)
	if err != nil {
		l.Error("HTTP shutdown failed", zap.Error(err))
	}

	l.Info("Closing infrastructure connections")

	err = rabbit.Channel.Close()
	if err != nil {
		l.Error("Failed to close RabbitMQ channel", zap.Error(err))
	}

	err = rabbit.Conn.Close()
	if err != nil {
		l.Error("Failed to close RabbitMQ connection", zap.Error(err))
	}

	err = infra.Redis.Close()
	if err != nil {
		l.Error("Failed to close Redis", zap.Error(err))
	}

	if infra.DB != nil {
		infra.DB.Close()
	}

	l.Info("Gracuful shutdown complete.")
}

func initApp(ctx context.Context) (
	*zap.Logger, *rabbitmq.RabbitHandler,
	*database.Infrastructure, *worker.Client,
	config.ServerConfig,
) {
	logCfg, err := config.NewLoggerConfig(".env")

	if err != nil {
		log.Fatalf("Failed to load DB config: %v", err)
	}
	l := logger.NewLogger(&logCfg)

	serverCfg, err := config.NewServerConfig(".env")

	if err != nil {
		l.Error("Failed to load Server config: %v", zap.Error(err))
	}

	rmqCfg, err := config.NewRabbitMQConfig(".env")

	if err != nil {
		l.Error("Failed to load RabbitMQ config: %v", zap.Error(err))
	}

	rabbit, err := rabbitmq.NewRabbitHandler(rmqCfg.Dsn())

	err = rabbit.SetupQueues()
	if err != nil {
		l.Fatal("Failed to configure RabbitMQ topology", zap.Error(err))
	}

	if err != nil {
		l.Fatal("Failed to init RabbitMQ", zap.Error(err))
	}

	dbCfg, err := config.NewDBConfig(".env")
	if err != nil {
		l.Error("Failed to load DB config: %v", zap.Error(err))
	}

	infra, err := database.InitInfrastructure(ctx, dbCfg.Dsn())

	if err != nil {
		l.Fatal("Failed to init DB", zap.Error(err))
	}

	clientTarget := "localhost:50051"
	workerClient, err := worker.NewWorkerClient(clientTarget)
	if err != nil {
		l.Fatal("Failed to create worker client", zap.String("worker client target", clientTarget), zap.Error(err))
	}

	return l, rabbit, infra, workerClient, serverCfg
}

func setupRoutes(
	l *zap.Logger, rabbit *rabbitmq.RabbitHandler, infra *database.Infrastructure,
	workerClient *worker.Client,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	mux.HandleFunc("/process/status", func(w http.ResponseWriter, r *http.Request) {
		status, err := workerClient.GetStatus(r.Context())
		if err != nil {
			http.Error(w, "Worker unreachable: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Worker ID: %s\nStatus: %s\nActive jobs: %d\n",
			status.GetWorkerId(),
			status.GetStatus(),
			status.GetActiveJobs(),
		)
	})

	repo := database.NewJobRepository(infra)

	srv := &server.Server{
		Queue: rabbit,
		Repo:  repo,
		Log:   l,
	}

	mux.HandleFunc("/process", srv.HandleProcess)
	mux.HandleFunc("/ping", srv.HandlePing)

	return mux
}
