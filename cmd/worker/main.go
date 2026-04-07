package main

import (
	"context"
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
		log.Fatal(err)
	}

	logCfg := config.NewLoggerConfig()
	l := logger.NewLogger(&logCfg)

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

	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			l.Fatal("Failed to listen for gRPC", zap.Error(err))
		}

		s := grpc.NewServer()
		proto.RegisterWorkerServiceServer(s, &worker.GrpcServer{WorkerId: "worker-01", Log: l})

		l.Info("gRPC listening on :50051")

		err = s.Serve(lis)
		if err != nil {
			l.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		l.Info("Metrics available at :2112/metrics")

		err := http.ListenAndServe(":2112", nil)
		if err != nil {
			l.Fatal("Failed to listen for metrics", zap.Error(err))
		}
	}()

	<-ctx.Done()
	l.Info("Worker gracefully shutting down...")

	err = rabbit.Channel.Cancel(rabbitmq.ConsumerTag, false)
	if err != nil {
		l.Error("Failed to cancel consumer", zap.Error(err))
	}

	l.Info("Waiting for active tasks to complete...")
	wg.Wait()

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

	l.Info("Graceful shutdown complete.")
}
