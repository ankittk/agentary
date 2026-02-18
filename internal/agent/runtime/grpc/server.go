package grpc

import (
	"github.com/ankittk/agentary/internal/agent/runtime"
	pb "github.com/ankittk/agentary/internal/agent/runtime/grpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server wraps a runtime.Runtime and exposes it via gRPC.
type Server struct {
	pb.UnimplementedAgentRuntimeServer
	Runtime runtime.Runtime
}

// RunTurn runs one turn: receive request, call Runtime.RunTurn, stream events then result.
func (s *Server) RunTurn(req *pb.TurnRequest, stream pb.AgentRuntime_RunTurnServer) error {
	if s.Runtime == nil {
		return status.Error(codes.Internal, "runtime not set")
	}
	ctx := stream.Context()
	rReq := protoTurnRequest(req)
	result, err := s.Runtime.RunTurn(ctx, rReq, func(ev runtime.Event) {
		pe := eventToProto(ev)
		_ = stream.Send(&pb.RunTurnResponse{Msg: &pb.RunTurnResponse_Event{Event: pe}})
	})
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	return stream.Send(&pb.RunTurnResponse{Msg: &pb.RunTurnResponse_Result{Result: &pb.TurnResult{Output: result.Output}}})
}
