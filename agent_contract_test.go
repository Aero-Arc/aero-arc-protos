package protos_test

import (
	"testing"

	agentv1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
	"google.golang.org/protobuf/proto"
)

func TestTelemetryFrameWALGenerationIdentity(t *testing.T) {
	frame := &agentv1.TelemetryFrame{AgentId: "agent-1", Seq: 42, WalId: "0195f6a8-86d1-7be7-a104-3a814dc19f9e"}
	field := frame.ProtoReflect().Descriptor().Fields().ByName("wal_id")
	if field == nil || field.Number() != 14 {
		t.Fatalf("wal_id descriptor = %#v, want field number 14", field)
	}
	wire, err := proto.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &agentv1.TelemetryFrame{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(frame, decoded) {
		t.Fatalf("decoded frame = %#v, want %#v", decoded, frame)
	}
}
