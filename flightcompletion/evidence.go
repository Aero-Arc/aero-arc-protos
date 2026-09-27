// Package flightcompletion validates immutable aircraft completion notifications.
package flightcompletion

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	pb "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
	"google.golang.org/protobuf/proto"
	"strings"
	"time"
)

// Validate checks binding and the capture-time evidence sequence. Delivery may
// occur much later; receipt freshness must not invalidate offline evidence.
//
// Parameters: e is immutable completion evidence bound to one Agent, exact
// operation context, canonical lowercase mission digest, and start command.
// Returns: nil for a supported outcome and chronologically valid airborne,
// terminal, and contemporaneous landed/disarmed observations; an error for nil,
// incomplete, oversized, noncanonical, unsupported, or inconsistent evidence.
// Validation does not authenticate the producer or authorize flight finalization.
func Validate(e *pb.FlightCompletionEvidence) error {
	if e == nil || e.EventId == "" || e.AgentId == "" || e.Context == nil || e.Context.AircraftId == "" || e.Context.FlightId == "" || e.Context.IntentId == "" || e.Context.IntentVersion == 0 || e.MissionId == "" || len(e.MissionDigest) != 64 || e.StartCommandId == "" || e.ObservationEpoch == "" {
		return fmt.Errorf("complete flight evidence binding required")
	}
	if _, err := hex.DecodeString(e.MissionDigest); err != nil || e.MissionDigest != strings.ToLower(e.MissionDigest) {
		return fmt.Errorf("invalid mission digest")
	}
	if e.Outcome != "mission_completed" && e.Outcome != "ended_early" {
		return fmt.Errorf("invalid completion outcome")
	}
	if e.AirborneAtUnixNs <= 0 || e.TerminalAtUnixNs < e.AirborneAtUnixNs || e.LandedAtUnixNs < e.TerminalAtUnixNs || e.DisarmedAtUnixNs < e.TerminalAtUnixNs {
		return fmt.Errorf("invalid completion evidence order")
	}
	delta := e.LandedAtUnixNs - e.DisarmedAtUnixNs
	if delta > int64(5*time.Second) || delta < -int64(5*time.Second) {
		return fmt.Errorf("landed and disarmed observations are not contemporaneous")
	}
	if len(e.EventId) > 128 || len(e.AgentId) > 128 || len(e.Context.FlightId) > 128 {
		return fmt.Errorf("completion identity too long")
	}
	return nil
}

// Encode returns deterministic payload bytes and a receipt digest after validation.
//
// Parameters: e is the unmodified completion evidence accepted by Validate.
// Returns: deterministic protobuf bytes, their lowercase SHA-256 digest, and nil
// on success. Validation or protobuf marshaling failures return nil bytes, an
// empty digest, and the error; callers must not acknowledge such evidence. The
// receipt digest binds the encoded event, not just its mission or event identity.
func Encode(e *pb.FlightCompletionEvidence) ([]byte, string, error) {
	if err := Validate(e); err != nil {
		return nil, "", err
	}
	raw, err := (proto.MarshalOptions{Deterministic: true}).Marshal(e)
	if err != nil {
		return nil, "", err
	}
	h := sha256.Sum256(raw)
	return raw, hex.EncodeToString(h[:]), nil
}
