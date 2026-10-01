// Package commanddigest defines the versioned, transport-independent command identity.
package commanddigest

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"unicode/utf8"

	pb "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
	"github.com/aero-arc/aero-arc-protos/missiondigest"
)

// Digest validates c and hashes canonical version-one bytes. Field order,
// integer millisecond timestamps, IEEE float32 bits, and domain prefix are fixed.
// The command ID, transport routing, digest field, and attempt metadata are not
// digest inputs. Agent identity is an immutable target, not Relay placement.
// Unsupported fields and nonfinite numbers are rejected rather than ignored.
func Digest(c *pb.DurableCommand) (string, error) {
	if c == nil || c.OperatorId == "" || c.AircraftId == "" || c.AgentId == "" || c.Definition == "" || c.DefinitionVersion == 0 || c.Context == nil || c.Context.AircraftId != c.AircraftId || c.Context.FlightId == "" || c.Context.IntentId == "" || c.Context.IntentVersion == 0 || c.IssuedAtUnixMs <= 0 || c.ExpiresAtUnixMs <= c.IssuedAtUnixMs || c.ExpiresAtUnixMs-c.IssuedAtUnixMs > 300000 || (c.RecoveryPolicy != "no_repeat_effect_v1" && c.RecoveryPolicy != "mission_readback_v1") {
		return "", fmt.Errorf("invalid command authority or recovery policy")
	}
	if len(c.ProtoReflect().GetUnknown()) != 0 || len(c.Context.ProtoReflect().GetUnknown()) != 0 {
		return "", fmt.Errorf("unknown command fields")
	}
	type wire struct {
		Operator, Aircraft, Agent, Flight, Intent                string
		IntentVersion                                            uint32
		Definition                                               string
		Version                                                  uint32
		Capability                                               string
		Issued, Expires                                          int64
		Recovery                                                 string
		Command                                                  uint32
		Parameters                                               []uint32
		Int                                                      bool
		Frame                                                    uint32
		X, Y                                                     int32
		Z                                                        uint32
		Profile                                                  string
		Observation                                              string
		Mode                                                     uint32
		MissionDigest, MissionID, DeploymentID, MissionCommandID string
		MissionVersion                                           uint32
	}
	w := wire{Operator: c.OperatorId, Aircraft: c.AircraftId, Agent: c.AgentId, Flight: c.Context.FlightId, Intent: c.Context.IntentId, IntentVersion: c.Context.IntentVersion, Definition: c.Definition, Version: c.DefinitionVersion, Capability: c.Capability, Issued: c.IssuedAtUnixMs, Expires: c.ExpiresAtUnixMs, Recovery: c.RecoveryPolicy}
	if m := c.GetMavlink(); m != nil {
		if c.RecoveryPolicy != "no_repeat_effect_v1" || c.Capability != "mavlink_command_v1" || m.Command == 0 || m.Command > 65535 || len(m.Parameters) != 7 || m.Frame > 255 || len(m.ProtoReflect().GetUnknown()) != 0 {
			return "", fmt.Errorf("unsupported MAVLink envelope")
		}
		if (!m.UseCommandInt && (m.Frame != 0 || m.X != 0 || m.Y != 0 || m.Z != 0)) || (m.UseCommandInt && (m.Parameters[4] != 0 || m.Parameters[5] != 0 || m.Parameters[6] != 0)) {
			return "", fmt.Errorf("unused MAVLink encoding fields must be zero")
		}
		for _, p := range m.Parameters {
			if math.IsNaN(float64(p)) || math.IsInf(float64(p), 0) || (p == 0 && math.Signbit(float64(p))) {
				return "", fmt.Errorf("parameters must be finite canonical float32")
			}
			w.Parameters = append(w.Parameters, math.Float32bits(p))
		}
		if math.IsNaN(float64(m.Z)) || math.IsInf(float64(m.Z), 0) || (m.Z == 0 && math.Signbit(float64(m.Z))) {
			return "", fmt.Errorf("invalid z")
		}
		w.Profile = m.VehicleProfile
		if m.MissionPrecondition != nil {
			digest, err := validatedMissionDigest(m.MissionPrecondition)
			if err != nil {
				return "", err
			}
			if m.MissionPreconditionId == "" || m.MissionPreconditionVersion == 0 {
				return "", fmt.Errorf("mission precondition identity required")
			}
			w.MissionDigest = digest
			w.MissionID = m.MissionPreconditionId
			w.MissionVersion = m.MissionPreconditionVersion
		} else if m.MissionPreconditionId != "" || m.MissionPreconditionVersion != 0 {
			return "", fmt.Errorf("mission precondition missing")
		}
		w.Command = m.Command
		w.Int = m.UseCommandInt
		w.Frame = m.Frame
		w.X = m.X
		w.Y = m.Y
		w.Z = math.Float32bits(m.Z)
		w.Observation = m.Observation
		w.Mode = m.ExpectedCustomMode
	} else if m := c.GetMission(); m != nil {
		if c.RecoveryPolicy != "mission_readback_v1" || c.Capability != "mission_upload_v1" || m.Binding == nil || len(m.ProtoReflect().GetUnknown()) != 0 || len(m.Binding.ProtoReflect().GetUnknown()) != 0 {
			return "", fmt.Errorf("invalid mission envelope")
		}
		b := m.Binding
		if b.MissionId == "" || b.DeploymentId == "" || b.MissionVersion == 0 || m.CommandId == "" || b.OperatorId != c.OperatorId || b.AircraftId != c.AircraftId || b.FlightId != w.Flight || b.IntentId != w.Intent || b.IntentVersion != w.IntentVersion || m.IssuedAtUnixMs != c.IssuedAtUnixMs || m.ExpiresAtUnixMs != c.ExpiresAtUnixMs {
			return "", fmt.Errorf("mission binding mismatch")
		}
		digest, err := validatedMissionDigest(m.Plan)
		if err != nil {
			return "", err
		}
		if digest != b.MissionDigest {
			return "", fmt.Errorf("mission digest mismatch")
		}
		w.MissionDigest = digest
		w.MissionID = b.MissionId
		w.MissionVersion = b.MissionVersion
		w.DeploymentID = b.DeploymentId
		w.MissionCommandID = m.CommandId
	} else {
		return "", fmt.Errorf("execution required")
	}
	if !utf8.ValidString(c.CommandId) || !utf8.ValidString(c.CommandDigest) {
		return "", fmt.Errorf("command identity must be UTF-8")
	}
	data := []byte("aeroarc-command-v1\x00")
	// Strings use UTF-8 byte lengths; integers are fixed-width network byte order.
	// Float values above were converted to IEEE-754 float32 bit patterns.
	for _, value := range []any{w.Operator, w.Aircraft, w.Agent, w.Flight, w.Intent, w.IntentVersion, w.Definition, w.Version, w.Capability, w.Issued, w.Expires, w.Recovery, w.Command, w.Parameters, w.Int, w.Frame, w.X, w.Y, w.Z, w.Profile, w.Observation, w.Mode, w.MissionDigest, w.MissionID, w.DeploymentID, w.MissionCommandID, w.MissionVersion} {
		switch v := value.(type) {
		case string:
			if !utf8.ValidString(v) {
				return "", fmt.Errorf("command strings must be UTF-8")
			}
			data = binary.BigEndian.AppendUint32(data, uint32(len(v)))
			data = append(data, []byte(v)...)
		case uint32:
			data = binary.BigEndian.AppendUint32(data, v)
		case int32:
			data = binary.BigEndian.AppendUint32(data, uint32(v))
		case int64:
			data = binary.BigEndian.AppendUint64(data, uint64(v))
		case bool:
			if v {
				data = append(data, 1)
			} else {
				data = append(data, 0)
			}
		case []uint32:
			data = binary.BigEndian.AppendUint32(data, uint32(len(v)))
			for _, p := range v {
				data = binary.BigEndian.AppendUint32(data, p)
			}
		}
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// validatedMissionDigest enforces schema-one adapter semantics before hashing.
// missiondigest itself deliberately remains a structural cross-runtime encoder.
func validatedMissionDigest(plan *pb.MissionPlan) (string, error) {
	digest, err := missiondigest.Digest(plan)
	if err != nil {
		return "", err
	}
	zero := func(v float64) bool { return v == 0 && !math.Signbit(v) }
	for i, item := range plan.Items {
		if item.Sequence != uint32(i) || item.Frame != 0 || item.Current || !item.Autocontinue {
			return "", fmt.Errorf("invalid mission item sequence, frame or flags at %d", i)
		}
		switch item.Command {
		case 16, 20, 21, 22:
		default:
			return "", fmt.Errorf("unsupported mission command at %d", i)
		}
		if item.Command == 20 && (i != len(plan.Items)-1 || item.LatitudeE7 != 0 || item.LongitudeE7 != 0 || item.AltitudeM != 0) {
			return "", fmt.Errorf("RTL must be terminal with zero coordinates")
		}
		if !zero(item.Param1) || !zero(item.Param2) || !zero(item.Param3) || (item.Command == 21 && item.Param4 != 1) || (item.Command != 21 && !zero(item.Param4)) {
			return "", fmt.Errorf("noncanonical mission parameters at %d", i)
		}
		altitude := float64(item.AltitudeM)
		if math.IsNaN(altitude) || math.IsInf(altitude, 0) || (altitude == 0 && math.Signbit(altitude)) || item.LatitudeE7 < -900000000 || item.LatitudeE7 > 900000000 || item.LongitudeE7 < -1800000000 || item.LongitudeE7 > 1800000000 {
			return "", fmt.Errorf("invalid mission coordinates at %d", i)
		}
		cm := item.AltitudeM * 100
		if float64(cm) < math.MinInt32 || float64(cm) > math.MaxInt32 || math.Float32bits(float32(int32(cm))*float32(0.01)) != math.Float32bits(item.AltitudeM) {
			return "", fmt.Errorf("mission altitude does not round-trip through centimeter storage at %d", i)
		}
	}
	return digest, nil
}
