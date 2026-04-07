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

	<-ctx.Done()
	l.Info("Gracefully shutting down...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(serverCfg.ReadHeaderTimeoutSec)*time.Second,
	)
	defer cancel()

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", serverCfg.Port),
		ReadHeaderTimeout: time.Duration(serverCfg.ReadHeaderTimeoutSec) * time.Second,
		Handler:           http.DefaultServeMux,
	}

	setupRoutes(l, rabbit, infra, workerClient, &serverCfg, httpServer)

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
	logCfg := config.NewLoggerConfig()
	l := logger.NewLogger(&logCfg)

	serverCfg := config.NewServerConfig(l)

	rmqCfg := config.NewRabbitMQConfig()
	rabbit, err := rabbitmq.NewRabbitHandler(rmqCfg.Dsn())

	if err != nil {
		l.Fatal("Failed to init RabbitMQ", zap.Error(err)) // TODO log
	}

	dbCfg := config.NewDBConfig()
	infra, err := database.InitInfrastructure(ctx, dbCfg.Dsn())

	if err != nil {
		l.Fatal("Failed to init DB", zap.Error(err)) // TODO log
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
	workerClient *worker.Client, serverCfg *config.ServerConfig, httpServer *http.Server,
) {
	http.Handle("/swagger/", httpSwagger.WrapHandler)

	http.HandleFunc("/process/status", func(w http.ResponseWriter, r *http.Request) {
		status, statusErr := workerClient.GetStatus(r.Context())

		if statusErr != nil {
			http.Error(w, "Worker unreachable: "+statusErr.Error(), http.StatusInternalServerError)

			return
		}

		fmt.Fprintf(w, "Worker ID: %s\nStatus: %s\nActive jobs: %d\n",
			status.GetWorkerId(), status.GetStatus(), status.GetActiveJobs(),
		)
	})

	repo := database.NewJobRepository(infra)

	srv := &server.Server{
		Queue: rabbit,
		Repo:  repo,
		Log:   l,
	}

	http.HandleFunc("/process", srv.HandleProcess)
	http.HandleFunc("/ping", srv.HandlePing)

	go func() {
		l.Info("Server starting", zap.String("port", httpServer.Addr))

		srv := &http.Server{
			Addr:         httpServer.Addr,
			Handler:      nil,
			ReadTimeout:  time.Duration(serverCfg.ReadTimeoutSec) * time.Second,
			WriteTimeout: time.Duration(serverCfg.WriteTimeoutSec) * time.Second,
			IdleTimeout:  time.Duration(serverCfg.IdleTimeoutSec) * time.Second,
		}

		err := srv.ListenAndServe()
		if err != nil {
			l.Fatal("Failed to start server", zap.String("port", httpServer.Addr), zap.Error(err))
		}
	}()
}
