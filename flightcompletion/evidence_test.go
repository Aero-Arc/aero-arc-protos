package flightcompletion

import (
	"bytes"
	"encoding/hex"
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

func TestCanonicalCompletionReceiptGolden(t *testing.T) {
	e := &pb.FlightCompletionEvidence{EventId: "event", AgentId: "agent", Context: &pb.OperationContext{FlightId: "flight", IntentId: "intent", IntentVersion: 300, AircraftId: "aircraft"}, MissionId: "mission", MissionDigest: strings.Repeat("a", 64), StartCommandId: "start", Outcome: "mission_completed", AirborneAtUnixNs: 128, TerminalAtUnixNs: 256, LandedAtUnixNs: 512, DisarmedAtUnixNs: 513, ObservationEpoch: "epoch"}
	raw, digest, err := Encode(e)
	want, _ := hex.DecodeString("0a056576656e7412056167656e741a1d0a06666c696768741206696e74656e7418ac022208616972637261667422076d697373696f6e2a4061616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161616161320573746172743a116d697373696f6e5f636f6d706c65746564408001488002508004588104620565706f6368")
	if err != nil || !bytes.Equal(raw, want) || digest != "6c39cbb8e66d501d544c44dbfff68bfce80fcb65de1ad8b6955048b319e9d95a" {
		t.Fatalf("canonical vector: %x %s %v", raw, digest, err)
	}
	// Compatibility check only; runtime serialization does not define the format.
	legacy, err := (proto.MarshalOptions{Deterministic: true}).Marshal(e)
	if err != nil || !bytes.Equal(legacy, raw) {
		t.Fatalf("existing receipt bytes changed: %v", err)
	}
	for _, mutate := range []func(*pb.FlightCompletionEvidence){
		func(v *pb.FlightCompletionEvidence) { v.EventId = string([]byte{255}) },
		func(v *pb.FlightCompletionEvidence) { v.ProtoReflect().SetUnknown([]byte{0x68, 1}) },
		func(v *pb.FlightCompletionEvidence) { v.Context.ProtoReflect().SetUnknown([]byte{0x28, 1}) },
	} {
		invalid := proto.Clone(e).(*pb.FlightCompletionEvidence)
		mutate(invalid)
		if raw, digest, err := Encode(invalid); err == nil || raw != nil || digest != "" {
			t.Fatal("noncanonical evidence accepted")
		}
	}
}
