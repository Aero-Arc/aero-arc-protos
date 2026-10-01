// Package flightcompletion validates immutable aircraft completion notifications.
package flightcompletion

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	pb "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"strings"
	"time"
	"unicode/utf8"
)

// Validate checks binding and the capture-time evidence sequence. Delivery may
// occur much later; receipt freshness must not invalidate offline evidence.
//
// Parameters: e is immutable completion evidence bound to one Agent, exact
// operation context, canonical lowercase mission digest, and start command.
// Returns: nil for a supported outcome and chronologically valid airborne,
// terminal, and contemporaneous landed/disarmed observations; an error for nil,
// incomplete, oversized (128-byte identities or 4096-byte encoded payload), noncanonical, unsupported, or inconsistent evidence.
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
	if len(e.ProtoReflect().GetUnknown()) != 0 || len(e.Context.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unknown completion fields are not canonical")
	}
	if proto.Size(e) > 4096 {
		return fmt.Errorf("completion evidence exceeds 4096-byte limit")
	}
	for _, id := range []string{e.EventId, e.AgentId, e.Context.FlightId, e.Context.AircraftId, e.Context.IntentId, e.MissionId, e.StartCommandId, e.ObservationEpoch} {
		if !utf8.ValidString(id) {
			return fmt.Errorf("completion identity is not UTF-8")
		}
		if len(id) > 128 {
			return fmt.Errorf("completion identity too long")
		}
	}
	return nil
}

// Encode returns canonical version-1 protobuf wire bytes and a receipt digest.
// The encoding is specified in flightcompletion/README.md independently of any
// protobuf runtime and retains the bytes produced by the original Go producer
// for supported, known-field evidence.
//
// Parameters: e is immutable completion evidence with valid UTF-8 identities
// and no unknown fields, as required by Validate.
// Returns: canonical bytes, their lowercase SHA-256 digest, and nil on success;
// invalid evidence returns nil bytes, an empty digest, and a validation error.
// Callers must not acknowledge invalid evidence. The digest binds every field.
func Encode(e *pb.FlightCompletionEvidence) ([]byte, string, error) {
	if err := Validate(e); err != nil {
		return nil, "", err
	}
	var context []byte
	context = appendString(context, 1, e.Context.FlightId)
	context = appendString(context, 2, e.Context.IntentId)
	context = appendInteger(context, 3, uint64(e.Context.IntentVersion))
	context = appendString(context, 4, e.Context.AircraftId)
	var raw []byte
	raw = appendString(raw, 1, e.EventId)
	raw = appendString(raw, 2, e.AgentId)
	raw = protowire.AppendTag(raw, 3, protowire.BytesType)
	raw = protowire.AppendBytes(raw, context)
	raw = appendString(raw, 4, e.MissionId)
	raw = appendString(raw, 5, e.MissionDigest)
	raw = appendString(raw, 6, e.StartCommandId)
	raw = appendString(raw, 7, e.Outcome)
	raw = appendInteger(raw, 8, uint64(e.AirborneAtUnixNs))
	raw = appendInteger(raw, 9, uint64(e.TerminalAtUnixNs))
	raw = appendInteger(raw, 10, uint64(e.LandedAtUnixNs))
	raw = appendInteger(raw, 11, uint64(e.DisarmedAtUnixNs))
	raw = appendString(raw, 12, e.ObservationEpoch)
	h := sha256.Sum256(raw)
	return raw, hex.EncodeToString(h[:]), nil
}

func appendString(raw []byte, number protowire.Number, value string) []byte {
	if value == "" {
		return raw
	}
	return protowire.AppendString(protowire.AppendTag(raw, number, protowire.BytesType), value)
}

func appendInteger(raw []byte, number protowire.Number, value uint64) []byte {
	if value == 0 {
		return raw
	}
	return protowire.AppendVarint(protowire.AppendTag(raw, number, protowire.VarintType), value)
}
