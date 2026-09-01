// Package missiondigest implements the cross-runtime Aero Arc mission-plan
// digest encoding. It does not replace service-specific validation or error
// mapping; callers must validate all schema-one normalization rules before use.
package missiondigest

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	agentv1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
)

var (
	// ErrNilPlan indicates that no mission plan was supplied for encoding.
	ErrNilPlan = errors.New("mission plan is nil")
	// ErrUnsupportedSchemaVersion indicates that the plan does not use the
	// schema-one encoding implemented by this package.
	ErrUnsupportedSchemaVersion = errors.New("unsupported mission schema version")
	// ErrInvalidItemCount indicates that the plan is empty or exceeds the
	// schema-one maximum of 200 operational items.
	ErrInvalidItemCount = errors.New("mission plan must contain 1 to 200 items")
	// ErrUnknownFields indicates that the plan or an item contains protobuf
	// fields not defined by schema one.
	ErrUnknownFields = errors.New("mission plan contains unknown fields")
	// ErrNilItem indicates that a repeated MissionPlan item is nil.
	ErrNilItem = errors.New("mission plan contains a nil item")
)

const schemaOnePrefix = "aeroarc-mission-plan-v1"

// CanonicalBytes returns a newly allocated schema-one canonical encoding of
// plan. The caller must first enforce the MissionPlan and MissionItem semantic
// normalization rules; this function enforces only structural preconditions
// needed to avoid ambiguous encoding. It returns ErrNilPlan,
// ErrUnsupportedSchemaVersion, ErrInvalidItemCount, ErrUnknownFields, or
// ErrNilItem (possibly wrapped with item context) when encoding is not defined.
func CanonicalBytes(plan *agentv1.MissionPlan) ([]byte, error) {
	if plan == nil {
		return nil, ErrNilPlan
	}
	if plan.GetSchemaVersion() != 1 {
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedSchemaVersion, plan.GetSchemaVersion())
	}
	if len(plan.GetItems()) == 0 || len(plan.GetItems()) > 200 {
		return nil, ErrInvalidItemCount
	}
	if len(plan.ProtoReflect().GetUnknown()) != 0 {
		return nil, ErrUnknownFields
	}

	const encodedItemBytes = 58
	canonical := make([]byte, 0, len(schemaOnePrefix)+1+4+encodedItemBytes*len(plan.GetItems()))
	canonical = append(canonical, schemaOnePrefix...)
	canonical = append(canonical, 0)
	canonical = binary.BigEndian.AppendUint32(canonical, uint32(len(plan.GetItems())))
	for index, item := range plan.GetItems() {
		if item == nil {
			return nil, fmt.Errorf("%w at index %d", ErrNilItem, index)
		}
		if len(item.ProtoReflect().GetUnknown()) != 0 {
			return nil, fmt.Errorf("%w at item %d", ErrUnknownFields, index)
		}
		canonical = binary.BigEndian.AppendUint32(canonical, item.GetSequence())
		canonical = binary.BigEndian.AppendUint32(canonical, item.GetFrame())
		canonical = binary.BigEndian.AppendUint32(canonical, item.GetCommand())
		canonical = append(canonical, boolByte(item.GetCurrent()), boolByte(item.GetAutocontinue()))
		canonical = binary.BigEndian.AppendUint64(canonical, math.Float64bits(item.GetParam1()))
		canonical = binary.BigEndian.AppendUint64(canonical, math.Float64bits(item.GetParam2()))
		canonical = binary.BigEndian.AppendUint64(canonical, math.Float64bits(item.GetParam3()))
		canonical = binary.BigEndian.AppendUint64(canonical, math.Float64bits(item.GetParam4()))
		canonical = binary.BigEndian.AppendUint32(canonical, uint32(item.GetLatitudeE7()))
		canonical = binary.BigEndian.AppendUint32(canonical, uint32(item.GetLongitudeE7()))
		canonical = binary.BigEndian.AppendUint32(canonical, math.Float32bits(item.GetAltitudeM()))
	}
	return canonical, nil
}

// Digest returns lowercase hexadecimal SHA-256 of the schema-one canonical
// bytes for plan. It returns the compatibility-sensitive CanonicalBytes errors
// unchanged, so callers may inspect them with errors.Is.
func Digest(plan *agentv1.MissionPlan) (string, error) {
	canonical, err := CanonicalBytes(plan)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func boolByte(value bool) byte {
	if value {
		return 1
	}
	return 0
}
