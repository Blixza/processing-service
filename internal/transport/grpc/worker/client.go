package worker

import (
	"context"
	"fmt"
	proto "main/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client proto.WorkerServiceClient
}

func NewWorkerClient(target string) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect to worker: %w", err)
	}

	return &Client{
		client: proto.NewWorkerServiceClient(conn),
	}, nil
}

func (c *Client) GetStatus(ctx context.Context) (*proto.StatusResponse, error) {
	return c.client.GetWorkerStatus(ctx, &proto.StatusRequest{})
}
