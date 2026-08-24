package protos_test

import (
	"testing"

	agentv1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestTelemetryFrameWALGenerationIdentity(t *testing.T) {
	frame := &agentv1.TelemetryFrame{
		AgentId: "agent-1",
		Seq:     42,
		WalId:   "0195f6a8-86d1-7be7-a104-3a814dc19f9e",
	}
	fields := frame.ProtoReflect().Descriptor().Fields()
	walIDField := fields.ByName("wal_id")
	if walIDField == nil ||
		walIDField.Number() != 14 ||
		walIDField.Kind() != protoreflect.StringKind {
		t.Fatalf("wal_id descriptor = %#v, want string field number 14", walIDField)
	}
	seqField := fields.ByName("seq")
	if seqField == nil ||
		seqField.Number() != 3 ||
		seqField.Kind() != protoreflect.Uint64Kind {
		t.Fatalf("seq descriptor = %#v, want uint64 field number 3", seqField)
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
