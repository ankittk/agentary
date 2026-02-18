// agentary-grpc-server runs the AgentRuntime gRPC server (stub runtime by default).
// Example: go run ./cmd/agentary-grpc-server --addr=:50051
// Then start the daemon with: agentary start --runtime=grpc --grpc-addr=localhost:50051
package main

import (
	"flag"
	"log"
	"net"

	agentrt "github.com/ankittk/agentary/internal/agent/runtime"
	agentgrpc "github.com/ankittk/agentary/internal/agent/runtime/grpc"
	pb "github.com/ankittk/agentary/internal/agent/runtime/grpc/pb"
	grpcgo "google.golang.org/grpc"
)

func main() {
	addr := flag.String("addr", ":50051", "gRPC listen address")
	flag.Parse()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	srv := grpcgo.NewServer()
	pb.RegisterAgentRuntimeServer(srv, &agentgrpc.Server{Runtime: agentrt.StubRuntime{}})
	log.Printf("AgentRuntime gRPC server listening on %s", *addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
