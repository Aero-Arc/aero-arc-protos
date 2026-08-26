package protos_test

import (
	"testing"

	agentv1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
	relayv1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/relay/v1"
	"google.golang.org/protobuf/proto"
)

func TestRelayDeployMissionRPCContract(t *testing.T) {
	service := relayv1.File_aeroarc_relay_v1_relay_proto.Services().ByName("RelayControl")
	method := service.Methods().ByName("DeployMission")
	if method == nil {
		t.Fatal("RelayControl.DeployMission descriptor is missing")
	}
	if got := method.Input().FullName(); got != "aeroarc.relay.v1.DeployMissionRequest" {
		t.Fatalf("DeployMission input = %q", got)
	}
	if got := method.Output().FullName(); got != "aeroarc.relay.v1.DeployMissionResponse" {
		t.Fatalf("DeployMission output = %q", got)
	}

	request := &relayv1.DeployMissionRequest{
		AgentId: "agent-1",
		Command: &agentv1.DeployMissionCommand{
			CommandId: "command-1",
			Binding: &agentv1.MissionBinding{
				MissionId: "mission-1", MissionVersion: 1, MissionDigest: "digest",
				DeploymentId: "deployment-1", OperatorId: "operator-1",
				AircraftId: "aircraft-1", FlightId: "flight-1",
				IntentId: "intent-1", IntentVersion: 1,
			},
			Plan: &agentv1.MissionPlan{SchemaVersion: 1},
		},
	}
	wire, err := proto.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &relayv1.DeployMissionRequest{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(request, decoded) {
		t.Fatalf("decoded request = %#v, want %#v", decoded, request)
	}
}
