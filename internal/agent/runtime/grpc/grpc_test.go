package grpc

import (
	"context"
	"testing"

	"github.com/ankittk/agentary/internal/agent/runtime"
	pb 	"github.com/ankittk/agentary/internal/agent/runtime/grpc/pb"
)

func TestServer_nilRuntime_returnsError(t *testing.T) {
	srv := &Server{Runtime: nil}
	req := &pb.TurnRequest{Input: "test"}
	stream := &mockRunTurnServer{ctx: context.Background()}
	err := srv.RunTurn(req, stream)
	if err == nil {
		t.Fatal("expected error when runtime is nil")
	}
}

func TestServer_stubRuntime_streamsResult(t *testing.T) {
	srv := &Server{Runtime: runtime.StubRuntime{}}
	req := &pb.TurnRequest{Input: "hello", Team: "t1", Agent: "a1"}
	stream := &mockRunTurnServer{ctx: context.Background()}
	err := srv.RunTurn(req, stream)
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if len(stream.sent) == 0 {
		t.Fatal("expected at least one response")
	}
	last := stream.sent[len(stream.sent)-1]
	if res := last.GetResult(); res == nil {
		t.Fatalf("expected final result, got %d messages", len(stream.sent))
	} else if res.Output != "stub: ok" {
		t.Errorf("result output: %q", res.Output)
	}
}

type mockRunTurnServer struct {
	pb.AgentRuntime_RunTurnServer
	ctx  context.Context
	sent []*pb.RunTurnResponse
}

func (m *mockRunTurnServer) Context() context.Context { return m.ctx }
func (m *mockRunTurnServer) Send(resp *pb.RunTurnResponse) error {
	m.sent = append(m.sent, resp)
	return nil
}
