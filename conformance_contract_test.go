package protos_test

import (
	"testing"

	conformancev1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/conformance/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestArmAssignmentContract(t *testing.T) {
	service := conformancev1.File_aeroarc_conformance_v1_conformance_proto.Services().ByName("ConformanceService")
	method := service.Methods().ByName("ArmAssignment")
	if method == nil {
		t.Fatal("ConformanceService method ArmAssignment is missing")
	}
	if got, want := method.Input().FullName(), protoreflect.FullName("aeroarc.conformance.v1.ArmAssignmentRequest"); got != want {
		t.Fatalf("ArmAssignment input = %q, want %q", got, want)
	}
	if got, want := method.Output().FullName(), protoreflect.FullName("aeroarc.conformance.v1.ArmAssignmentResponse"); got != want {
		t.Fatalf("ArmAssignment output = %q, want %q", got, want)
	}

	request := &conformancev1.ArmAssignmentRequest{
		Source:               "api",
		MessageId:            "arm-1",
		AssignmentId:         "assignment-1",
		AssignmentGeneration: 7,
	}
	wire, err := proto.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &conformancev1.ArmAssignmentRequest{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(request, decoded) {
		t.Fatalf("decoded request = %#v, want %#v", decoded, request)
	}
}
