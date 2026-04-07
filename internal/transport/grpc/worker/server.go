package worker

import (
	"context"
	"fmt"
	worker "main/proto"
	"time"

	"go.uber.org/zap"
)

type GrpcServer struct {
	worker.UnimplementedWorkerServiceServer
	WorkerId string
	Log      *zap.Logger
}

func (s *GrpcServer) GetWorkerStatus(ctx context.Context, req *worker.StatusRequest) (*worker.StatusResponse, error) {
	return &worker.StatusResponse{
		WorkerId:   s.WorkerId,
		Status:     "ACTIVE",
		ActiveJobs: 1, // TODO
	}, nil
}

func (s *GrpcServer) StreamJobLogs(req *worker.LogRequest, stream worker.WorkerService_StreamJobLogsServer) error {
	s.Log.Info("Streaming logs for job", zap.String("job id", req.JobId)) // TODO log

	for i := 1; i <= 5; i++ {
		resp := &worker.LogResponse{
			Line:               fmt.Sprintf("Processing chunk %d of 5...", i),
			ProgressPercentage: int32(i * 20),
			Timestamp:          time.Now().Format(time.RFC3339),
		}

		err := stream.Send(resp)
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}
