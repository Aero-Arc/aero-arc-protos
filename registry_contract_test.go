package protos_test

import (
	"testing"

	registryv1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/registry/v1"
	"google.golang.org/protobuf/proto"
)

func TestHeartbeatAgentRequestRelayOwnershipField(t *testing.T) {
	request := &registryv1.HeartbeatAgentRequest{
		AgentId:        "agent-1",
		TimestampUnixMs: 1_786_468_800_000,
		RelayId:        "relay-2",
	}

	field := request.ProtoReflect().Descriptor().Fields().ByName("relay_id")
	if field == nil || field.Number() != 3 {
		t.Fatalf("relay_id descriptor = %#v, want field number 3", field)
	}
	wire, err := proto.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &registryv1.HeartbeatAgentRequest{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetAgentId() != request.GetAgentId() || decoded.GetTimestampUnixMs() != request.GetTimestampUnixMs() || decoded.GetRelayId() != request.GetRelayId() {
		t.Fatalf("decoded request = %#v, want %#v", decoded, request)
	}
}
