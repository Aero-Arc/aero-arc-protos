package commanddigest

import (
	"math"
	"testing"

	pb "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
	"google.golang.org/protobuf/proto"
)

func sample() *pb.DurableCommand {
	return &pb.DurableCommand{CommandId: "command", OperatorId: "operator", AircraftId: "aircraft", AgentId: "agent", Context: &pb.OperationContext{AircraftId: "aircraft", FlightId: "flight", IntentId: "intent", IntentVersion: 1}, Definition: "ARM", DefinitionVersion: 1, Capability: "mavlink_command_v1", IssuedAtUnixMs: 1, ExpiresAtUnixMs: 30001, RecoveryPolicy: "no_repeat_effect_v1", Execution: &pb.DurableCommand_Mavlink{Mavlink: &pb.MavlinkExecution{Command: 400, Parameters: []float32{1, 0, 0, 0, 0, 0, 0}, Observation: "armed", VehicleProfile: "arducopter_v1"}}}
}
func TestDigestIndependentOfWireAndTransportIdentity(t *testing.T) {
	c := sample()
	want, err := Digest(c)
	if err != nil {
		t.Fatal(err)
	}
	// Independently encoded using network-order fields, not protobuf serialization.
	if want != "c99fff59b84a620116941115acd3e13737c306dec5a431dd4d25e60462d6f6bc" {
		t.Fatalf("canonical v1 compatibility changed: %s", want)
	}
	data, _ := proto.Marshal(c)
	var restored pb.DurableCommand
	if err = proto.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.CommandId = "new-id"
	restored.CommandDigest = "not-an-input"
	got, err := Digest(&restored)
	if err != nil || got != want {
		t.Fatalf("digest changed: %s %v", got, err)
	}
	restored.AgentId = "another-agent"
	got, _ = Digest(&restored)
	if got == want {
		t.Fatal("target change did not change digest")
	}
}
func TestDigestRejectsAmbiguousOrUnknownPayload(t *testing.T) {
	for name, mutate := range map[string]func(*pb.DurableCommand){"nan": func(c *pb.DurableCommand) { c.GetMavlink().Parameters[0] = float32(math.NaN()) }, "negative-zero": func(c *pb.DurableCommand) { c.GetMavlink().Parameters[0] = float32(math.Copysign(0, -1)) }, "unknown": func(c *pb.DurableCommand) { c.ProtoReflect().SetUnknown([]byte{0xf8, 0x07, 1}) }, "expiry": func(c *pb.DurableCommand) { c.ExpiresAtUnixMs = c.IssuedAtUnixMs }, "binding": func(c *pb.DurableCommand) { c.Context.AircraftId = "other" }} {
		t.Run(name, func(t *testing.T) {
			c := sample()
			mutate(c)
			if _, err := Digest(c); err == nil {
				t.Fatal("invalid canonical command accepted")
			}
		})
	}
}
