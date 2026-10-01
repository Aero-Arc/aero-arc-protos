package flightcompletion

import (
	pb "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
	"google.golang.org/protobuf/proto"
	"strings"
	"testing"
	"time"
)

func TestCompletionEvidenceRequiresBoundFreshGroundPair(t *testing.T) {
	at := time.Now().Add(-time.Hour).UnixNano()
	e := &pb.FlightCompletionEvidence{EventId: "event", AgentId: "agent", Context: &pb.OperationContext{AircraftId: "aircraft", FlightId: "flight", IntentId: "intent", IntentVersion: 1}, MissionId: "mission", MissionDigest: strings.Repeat("a", 64), StartCommandId: "start", Outcome: "mission_completed", AirborneAtUnixNs: at, TerminalAtUnixNs: at + int64(time.Minute), LandedAtUnixNs: at + int64(2*time.Minute), DisarmedAtUnixNs: at + int64(2*time.Minute), ObservationEpoch: "epoch"}
	if err := Validate(e); err != nil {
		t.Fatal(err)
	}
	upper := proto.Clone(e).(*pb.FlightCompletionEvidence)
	upper.MissionDigest = strings.ToUpper(upper.MissionDigest)
	if Validate(upper) == nil {
		t.Fatal("noncanonical mission digest accepted")
	}
	for _, change := range []func(*pb.FlightCompletionEvidence){
		func(v *pb.FlightCompletionEvidence) { v.MissionId = strings.Repeat("x", 129) },
		func(v *pb.FlightCompletionEvidence) { v.StartCommandId = strings.Repeat("x", 129) },
		func(v *pb.FlightCompletionEvidence) { v.ObservationEpoch = strings.Repeat("x", 129) },
		func(v *pb.FlightCompletionEvidence) { v.Context.AircraftId = strings.Repeat("x", 129) },
		func(v *pb.FlightCompletionEvidence) { v.Context.IntentId = strings.Repeat("x", 129) },
		func(v *pb.FlightCompletionEvidence) {
			v.ProtoReflect().SetUnknown(append([]byte{0xfa, 0x07, 0x80, 0x20}, []byte(strings.Repeat("x", 4096))...))
		},
	} {
		oversized := proto.Clone(e).(*pb.FlightCompletionEvidence)
		change(oversized)
		if _, _, err := Encode(oversized); err == nil {
			t.Fatal("oversized evidence accepted")
		}
	}
	raw, digest, err := Encode(e)
	if err != nil || len(raw) == 0 || len(digest) != 64 {
		t.Fatalf("encode %v", err)
	}
	stale := proto.Clone(e).(*pb.FlightCompletionEvidence)
	stale.DisarmedAtUnixNs += int64(6 * time.Second)
	if Validate(stale) == nil {
		t.Fatal("stale evidence accepted")
	}
	changed := proto.Clone(e).(*pb.FlightCompletionEvidence)
	changed.Outcome = "ended_early"
	_, other, err := Encode(changed)
	if err != nil || other == digest {
		t.Fatal("receipt fails to bind outcome")
	}
}
