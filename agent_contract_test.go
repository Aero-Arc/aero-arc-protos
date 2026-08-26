package protos_test

import (
	"crypto/sha256"
	"encoding/hex"
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

func TestMissionDeploymentContractRoundTrip(t *testing.T) {
	plan := &agentv1.MissionPlan{
		SchemaVersion: 1,
		Items: []*agentv1.MissionItem{{
			Sequence: 0, Frame: 3, Command: 16, Current: true, Autocontinue: true,
			LatitudeE7: -353632620, LongitudeE7: 1491652370, AltitudeM: 20,
		}},
	}
	canonical, err := proto.MarshalOptions{Deterministic: true}.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	binding := &agentv1.MissionBinding{
		MissionId: "mission-1", MissionVersion: 1,
		MissionDigest: hex.EncodeToString(digest[:]), DeploymentId: "deployment-1",
		OperatorId: "operator-1", AircraftId: "aircraft-1", FlightId: "flight-1",
		IntentId: "intent-1", IntentVersion: 2,
	}
	command := &agentv1.DeployMissionCommand{
		CommandId: "command-1", Binding: binding, Plan: plan,
		IssuedAtUnixMs: 1000, ExpiresAtUnixMs: 2000,
	}
	message := &agentv1.RelayStreamMessage{
		Payload: &agentv1.RelayStreamMessage_DeployMission{DeployMission: command},
	}
	wire, err := proto.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &agentv1.RelayStreamMessage{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(message, decoded) {
		t.Fatalf("decoded deployment = %#v, want %#v", decoded, message)
	}
	if got := decoded.GetDeployMission().GetBinding().GetMissionDigest(); got != hex.EncodeToString(digest[:]) {
		t.Fatalf("mission digest = %q, want %q", got, hex.EncodeToString(digest[:]))
	}

	relayPayload := (&agentv1.RelayStreamMessage{}).ProtoReflect().Descriptor().Oneofs().ByName("payload")
	deployField := relayPayload.Fields().ByName("deploy_mission")
	if deployField == nil || deployField.Number() != 5 || deployField.Message().Name() != "DeployMissionCommand" {
		t.Fatalf("deploy_mission descriptor = %#v, want field 5 DeployMissionCommand", deployField)
	}
	agentPayload := (&agentv1.AgentStreamMessage{}).ProtoReflect().Descriptor().Oneofs().ByName("payload")
	resultField := agentPayload.Fields().ByName("mission_deployment_result")
	if resultField == nil || resultField.Number() != 4 || resultField.Message().Name() != "MissionDeploymentResult" {
		t.Fatalf("mission_deployment_result descriptor = %#v, want field 4 MissionDeploymentResult", resultField)
	}
}

func TestMissionDeploymentResultStatusesRemainDistinct(t *testing.T) {
	want := map[agentv1.MissionDeploymentResult_Status]int32{
		agentv1.MissionDeploymentResult_STATUS_APPLIED:                  1,
		agentv1.MissionDeploymentResult_STATUS_ALREADY_APPLIED:          2,
		agentv1.MissionDeploymentResult_STATUS_REJECTED:                 3,
		agentv1.MissionDeploymentResult_STATUS_TEMPORARY_ERROR:          4,
		agentv1.MissionDeploymentResult_STATUS_OUTCOME_UNKNOWN:          5,
		agentv1.MissionDeploymentResult_STATUS_BINDING_MISMATCH:         6,
		agentv1.MissionDeploymentResult_STATUS_ONBOARD_MISSION_MISMATCH: 7,
	}
	for status, number := range want {
		if got := int32(status); got != number {
			t.Fatalf("%s = %d, want %d", status, got, number)
		}
	}
}
