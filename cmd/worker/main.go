package main

import (
	"context"
	"fmt"
	"log"
	"main/config"
	"main/internal/database"
	"main/internal/logger"
	"main/internal/transport/grpc/worker"
	"main/internal/transport/rabbitmq"
	proto "main/proto"
	"net"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := godotenv.Load()
	if err != nil {
		log.Printf("Failed to load env: %v\n", err)
		return
	}

	logCfg := config.NewLoggerConfig()
	l := logger.NewLogger(&logCfg)

	serverCfg := config.NewServerConfig(l)

	dbCfg := config.NewDBConfig()

	infra, err := database.InitInfrastructure(ctx, dbCfg.Dsn())
	if err != nil {
		l.Fatal("Worker Infra Error", zap.Error(err))
	}

	rmqCfg := config.NewRabbitMQConfig()

	rabbit, err := rabbitmq.NewRabbitHandler(rmqCfg.Dsn())
	if err != nil {
		l.Fatal("Worker RabbitMQ Error", zap.Error(err))
	}

	defer rabbit.Conn.Close()

	l.Info("Worker started. Listening for messages...")

	var wg sync.WaitGroup

	err = rabbit.ConsumeJobs(ctx, l, infra, &wg)
	if err != nil {
		l.Fatal("Failed to start consumer", zap.Error(err))
	}

	go startGrpcServer(ctx, l, &serverCfg)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", serverCfg.MetricsPort),
		ReadHeaderTimeout: time.Duration(serverCfg.ReadHeaderTimeoutSec) * time.Second,
		Handler:           http.DefaultServeMux,
	}

	go func() {
		startMetricsServer(l, &serverCfg, httpServer)
	}()

	<-ctx.Done()
	l.Info("Worker gracefully shutting down...")

	err = rabbit.Channel.Cancel(rabbitmq.ConsumerTag, false)
	if err != nil {
		l.Error("Failed to cancel consumer", zap.Error(err))
	}

	l.Info("Waiting for active tasks to complete...")

	closeTasks(rabbit, infra, l)

	wg.Wait()

	l.Info("Graceful shutdown complete.")
}

func closeTasks(rabbit *rabbitmq.RabbitHandler, infra *database.Infrastructure, l *zap.Logger) {
	err := rabbit.Channel.Close()
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
}

func startGrpcServer(ctx context.Context, l *zap.Logger, serverCfg *config.ServerConfig) {
	grpcPortStr := fmt.Sprintf(":%d", serverCfg.GrpcPort)
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", grpcPortStr)
	if err != nil {
		l.Fatal("Failed to listen for gRPC", zap.Error(err))
	}

	s := grpc.NewServer()
	proto.RegisterWorkerServiceServer(s, &worker.GrpcServer{WorkerID: "worker-01", Log: l})

	l.Info(fmt.Sprintf("gRPC listening on %s", grpcPortStr))

	err = s.Serve(lis)
	if err != nil {
		l.Fatal("Failed to serve gRPC", zap.Error(err))
	}
}

func startMetricsServer(l *zap.Logger, serverCfg *config.ServerConfig, httpServer *http.Server) {
	http.Handle("/metrics", promhttp.Handler())
	metricsPortStr := fmt.Sprintf(":%d", serverCfg.MetricsPort)
	l.Info(fmt.Sprintf("Metrics available at %s/metrics", metricsPortStr))

	srv := &http.Server{
		Addr:         httpServer.Addr,
		Handler:      nil,
		ReadTimeout:  time.Duration(serverCfg.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(serverCfg.WriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(serverCfg.IdleTimeoutSec) * time.Second,
	}

	err := srv.ListenAndServe()
	if err != nil {
		l.Fatal("Failed to listen for metrics", zap.Error(err))
	}
}
