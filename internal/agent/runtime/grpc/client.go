package grpc

import (
	"context"

	"github.com/ankittk/agentary/internal/agent/runtime"
	pb "github.com/ankittk/agentary/internal/agent/runtime/grpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a runtime.Runtime that calls a gRPC AgentRuntime server.
type Client struct {
	// Addr is the gRPC server address (e.g. "localhost:50051").
	Addr string
	// DialOptions are used when connecting (e.g. TLS, interceptors).
	DialOptions []grpc.DialOption
}

// Name returns "grpc".
func (c *Client) Name() string { return "grpc" }

// RunTurn calls the gRPC server's RunTurn, streams events to emit, and returns the result.
func (c *Client) RunTurn(ctx context.Context, req runtime.TurnRequest, emit func(runtime.Event)) (runtime.TurnResult, error) {
	opts := c.DialOptions
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	conn, err := grpc.NewClient(c.Addr, opts...)
	if err != nil {
		return runtime.TurnResult{}, err
	}
	defer func() { _ = conn.Close() }()

	client := pb.NewAgentRuntimeClient(conn)
	preq := turnRequestToProto(req)
	stream, err := client.RunTurn(ctx, preq)
	if err != nil {
		return runtime.TurnResult{}, err
	}

	var result runtime.TurnResult
	for {
		resp, err := stream.Recv()
		if err != nil {
			return runtime.TurnResult{}, err
		}
		switch m := resp.Msg.(type) {
		case *pb.RunTurnResponse_Event:
			if m.Event != nil {
				emit(protoToEvent(m.Event))
			}
		case *pb.RunTurnResponse_Result:
			if m.Result != nil {
				result.Output = m.Result.GetOutput()
			}
			return result, nil
		default:
			// skip unknown
		}
	}
}

func turnRequestToProto(req runtime.TurnRequest) *pb.TurnRequest {
	preq := &pb.TurnRequest{
		Team:             req.Team,
		Agent:            req.Agent,
		Input:            req.Input,
		NetworkAllowlist: req.NetworkAllowlist,
		Model:            req.Model,
		MaxTokens:        int32(req.MaxTokens),
	}
	if req.TaskID != nil {
		preq.TaskId = req.TaskID
	}
	return preq
}
