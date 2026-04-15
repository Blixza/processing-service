package worker

import (
	"context"
	"fmt"
	proto "main/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client proto.WorkerServiceClient
}

func NewWorkerClient(target string) (*Client, error) {
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("could not connect to worker: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		state := conn.GetState()
		if state == connectivity.Ready || state == connectivity.Idle {
			break
		}

		if !conn.WaitForStateChange(ctx, state) {
			return nil, fmt.Errorf("worker gRPC service failed to connect within 10s (current state: %s)", state)
		}
	}

	return &Client{
		conn:   conn,
		client: proto.NewWorkerServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetStatus(ctx context.Context) (*proto.StatusResponse, error) {
	return c.client.GetWorkerStatus(ctx, &proto.StatusRequest{})
}
