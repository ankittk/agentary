package grpc

import (
	"github.com/ankittk/agentary/internal/agent/runtime"
	pb "github.com/ankittk/agentary/internal/agent/runtime/grpc/pb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func protoTurnRequest(req *pb.TurnRequest) runtime.TurnRequest {
	if req == nil {
		return runtime.TurnRequest{}
	}
	r := runtime.TurnRequest{
		Team:             req.GetTeam(),
		Agent:            req.GetAgent(),
		Input:            req.GetInput(),
		NetworkAllowlist: req.GetNetworkAllowlist(),
		Model:            req.GetModel(),
		MaxTokens:        int(req.GetMaxTokens()),
	}
	if req.TaskId != nil {
		r.TaskID = req.TaskId
	}
	return r
}

func eventToProto(ev runtime.Event) *pb.Event {
	pe := &pb.Event{
		Type:   ev.Type,
		Team:   ev.Team,
		Agent:  ev.Agent,
		TaskId: ev.TaskID,
	}
	if !ev.Timestamp.IsZero() {
		pe.Timestamp = timestamppb.New(ev.Timestamp)
	}
	if len(ev.Data) > 0 {
		m := make(map[string]interface{})
		for k, v := range ev.Data {
			m[k] = v
		}
		if st, err := structpb.NewStruct(m); err == nil {
			pe.Data = st
		}
	}
	return pe
}

func protoToEvent(pe *pb.Event) runtime.Event {
	if pe == nil {
		return runtime.Event{}
	}
	ev := runtime.Event{
		Type:   pe.GetType(),
		Team:   pe.GetTeam(),
		Agent:  pe.GetAgent(),
		TaskID: pe.TaskId,
	}
	if pe.Timestamp != nil {
		ev.Timestamp = pe.Timestamp.AsTime()
	}
	if pe.Data != nil {
		if m := pe.Data.AsMap(); m != nil {
			ev.Data = make(map[string]any)
			for k, v := range m {
				ev.Data[k] = v
			}
		}
	}
	return ev
}
