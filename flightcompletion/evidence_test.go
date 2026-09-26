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
