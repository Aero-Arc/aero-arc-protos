package commanddigest

import (
	"fmt"
	"github.com/aero-arc/aero-arc-protos/missiondigest"
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

func TestUnusedMAVLinkFieldsAreRejected(t *testing.T) {
	for _, change := range []func(*pb.MavlinkExecution){func(m *pb.MavlinkExecution) { m.X = 1 }, func(m *pb.MavlinkExecution) { m.Y = 1 }, func(m *pb.MavlinkExecution) { m.Z = 1 }, func(m *pb.MavlinkExecution) { m.Frame = 1 }, func(m *pb.MavlinkExecution) { m.UseCommandInt = true; m.Parameters[4] = 1 }} {
		c := sample()
		change(c.GetMavlink())
		if _, err := Digest(c); err == nil {
			t.Fatal("unused fields accepted")
		}
	}
}
func TestMissionSemanticsAndBindingsAreValidated(t *testing.T) {
	plan := &pb.MissionPlan{SchemaVersion: 1, Items: []*pb.MissionItem{{Command: 21, Autocontinue: true, Param4: 1}}}
	for _, precondition := range []bool{false, true} {
		for name, change := range map[string]func(*pb.MissionItem){"sequence": func(i *pb.MissionItem) { i.Sequence = 2 }, "frame": func(i *pb.MissionItem) { i.Frame = 3 }, "command": func(i *pb.MissionItem) { i.Command = 99 }, "autocontinue": func(i *pb.MissionItem) { i.Autocontinue = false }, "current": func(i *pb.MissionItem) { i.Current = true }, "latitude": func(i *pb.MissionItem) { i.LatitudeE7 = 900000001 }, "parameter": func(i *pb.MissionItem) { i.Param1 = math.NaN() }, "negativezero": func(i *pb.MissionItem) { i.Param2 = math.Copysign(0, -1) }, "altitude": func(i *pb.MissionItem) { i.AltitudeM = 16.8 }} {
			t.Run(fmt.Sprintf("%t/%s", precondition, name), func(t *testing.T) {
				p := proto.Clone(plan).(*pb.MissionPlan)
				change(p.Items[0])
				digest, _ := missiondigest.Digest(p)
				c := sample()
				if precondition {
					m := c.GetMavlink()
					m.MissionPrecondition = p
					m.MissionPreconditionId = "mission"
					m.MissionPreconditionVersion = 1
				} else {
					c.Capability = "mission_upload_v1"
					c.RecoveryPolicy = "mission_readback_v1"
					c.Execution = &pb.DurableCommand_Mission{Mission: &pb.DeployMissionCommand{CommandId: "deploy-command", Binding: &pb.MissionBinding{OperatorId: c.OperatorId, AircraftId: c.AircraftId, FlightId: c.Context.FlightId, IntentId: c.Context.IntentId, IntentVersion: 1, MissionId: "mission", MissionVersion: 1, DeploymentId: "deployment", MissionDigest: digest}, Plan: p, IssuedAtUnixMs: c.IssuedAtUnixMs, ExpiresAtUnixMs: c.ExpiresAtUnixMs}}
				}
				if _, err := Digest(c); err == nil {
					t.Fatal("invalid mission semantics accepted")
				}
			})
		}
	}
	digest, _ := missiondigest.Digest(plan)
	c := sample()
	c.Capability = "mission_upload_v1"
	c.RecoveryPolicy = "mission_readback_v1"
	c.Execution = &pb.DurableCommand_Mission{Mission: &pb.DeployMissionCommand{CommandId: "deploy-command", Binding: &pb.MissionBinding{OperatorId: c.OperatorId, AircraftId: c.AircraftId, FlightId: c.Context.FlightId, IntentId: c.Context.IntentId, IntentVersion: 1, MissionId: "mission", MissionVersion: 1, DeploymentId: "deployment", MissionDigest: digest}, Plan: plan, IssuedAtUnixMs: c.IssuedAtUnixMs, ExpiresAtUnixMs: c.ExpiresAtUnixMs}}
	if _, err := Digest(c); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*pb.MissionBinding){func(b *pb.MissionBinding) { b.MissionId = "" }, func(b *pb.MissionBinding) { b.DeploymentId = "" }, func(b *pb.MissionBinding) { b.MissionVersion = 0 }} {
		broken := proto.Clone(c).(*pb.DurableCommand)
		change(broken.GetMission().Binding)
		if _, err := Digest(broken); err == nil {
			t.Fatal("incomplete mission identity accepted")
		}
	}
}

func TestMissionAltitudeUsesArduPilotFloat32Readback(t *testing.T) {
	for _, tc := range []struct {
		altitude float32
		valid    bool
	}{
		{0.05, false}, {float32(5) * float32(0.01), false},
		{float32(9) * float32(0.01), true}, {0.09, false}, {20, true},
	} {
		plan := &pb.MissionPlan{SchemaVersion: 1, Items: []*pb.MissionItem{{Command: 16, Autocontinue: true, AltitudeM: tc.altitude}}}
		_, err := validatedMissionDigest(plan)
		if (err == nil) != tc.valid {
			t.Errorf("altitude %.9g valid=%v: %v", tc.altitude, tc.valid, err)
		}
	}
}
