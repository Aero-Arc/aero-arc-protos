package protos_test

import (
	"testing"

	conformancev1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/conformance/v1"
	registryv1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/registry/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestHeartbeatAgentRequestRelayOwnershipField(t *testing.T) {
	request := &registryv1.HeartbeatAgentRequest{
		AgentId:         "agent-1",
		TimestampUnixMs: 1_786_468_800_000,
		RelayId:         "relay-2",
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

func TestConformanceProjectionContracts(t *testing.T) {
	service := registryv1.File_aeroarc_registry_v1_registry_proto.Services().ByName("AeroRegistry")
	for _, method := range []string{"PublishConformanceSummary", "GetConformanceSummary", "BatchGetConformanceSummaries"} {
		if service.Methods().ByName(protoreflect.Name(method)) == nil {
			t.Fatalf("AeroRegistry method %q is missing", method)
		}
	}

	summary := &conformancev1.ConformanceSummary{
		AssignmentId:         "assignment-1",
		AssignmentGeneration: 7,
		EvaluationRevision:   11,
		EvaluationId:         "registry:assignment-1:7:11",
		Condition:            conformancev1.ConformanceCondition_CONFORMANCE_CONDITION_NON_CONFORMING,
		MonitoringStatus:     conformancev1.MonitoringStatus_MONITORING_STATUS_CURRENT,
		RecordingStatus:      conformancev1.RecordingStatus_RECORDING_STATUS_CONFIRMED,
	}
	wire, err := proto.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &conformancev1.ConformanceSummary{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(summary, decoded) {
		t.Fatalf("decoded summary = %#v, want %#v", decoded, summary)
	}
}
