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

	WorkerID string
	Log      *zap.Logger
}

func (s *GrpcServer) GetWorkerStatus(_ context.Context, _ *worker.StatusRequest) (*worker.StatusResponse, error) {
	return &worker.StatusResponse{
		WorkerId:   s.WorkerID,
		Status:     "ACTIVE",
		ActiveJobs: 1, // TODO
	}, nil
}

func (s *GrpcServer) StreamJobLogs(req *worker.LogRequest, stream worker.WorkerService_StreamJobLogsServer) error {
	s.Log.Info("Streaming logs for job", zap.String("job id", req.GetJobId())) // TODO log

	for i := 1; i <= 5; i++ {
		resp := &worker.LogResponse{
			Line:               fmt.Sprintf("Processing chunk %d of 5...", i),
			ProgressPercentage: int32(i * 20), //nolint:mnd // because divide 20, 40, 60, 80, 100 percents
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
