package missiondigest_test

import (
	"encoding/hex"
	"errors"
	"testing"

	agentv1 "github.com/aero-arc/aero-arc-protos/gen/go/aeroarc/agent/v1"
	"github.com/aero-arc/aero-arc-protos/missiondigest"
)

const (
	canonicalHex = "6165726f6172632d6d697373696f6e2d706c616e2d7631000000000100000000000000000000001000010000000000000000000000000000000000000000000000000000000000000000eaebfe9458e8cf1241a0cccd"
	digestHex    = "6efa96b36af29a800d53ee7d7baf57d4b24f00d9ce2b408327281e74824acf4f"
)

func TestSchemaOneGoldenVector(t *testing.T) {
	plan := goldenPlan()
	canonical, err := missiondigest.CanonicalBytes(plan)
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(canonical); got != canonicalHex {
		t.Fatalf("canonical bytes = %q, want %q", got, canonicalHex)
	}
	digest, err := missiondigest.Digest(plan)
	if err != nil {
		t.Fatal(err)
	}
	if digest != digestHex {
		t.Fatalf("digest = %q, want %q", digest, digestHex)
	}
}

func TestCanonicalBytesRejectsUndefinedInputs(t *testing.T) {
	tests := []struct {
		name string
		plan *agentv1.MissionPlan
		want error
	}{
		{name: "nil plan", want: missiondigest.ErrNilPlan},
		{name: "unsupported schema", plan: &agentv1.MissionPlan{SchemaVersion: 2}, want: missiondigest.ErrUnsupportedSchemaVersion},
		{name: "empty plan", plan: &agentv1.MissionPlan{SchemaVersion: 1}, want: missiondigest.ErrInvalidItemCount},
		{name: "nil item", plan: &agentv1.MissionPlan{SchemaVersion: 1, Items: []*agentv1.MissionItem{nil}}, want: missiondigest.ErrNilItem},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := missiondigest.CanonicalBytes(test.plan); !errors.Is(err, test.want) {
				t.Fatalf("CanonicalBytes() error = %v, want %v", err, test.want)
			}
		})
	}
}

func goldenPlan() *agentv1.MissionPlan {
	return &agentv1.MissionPlan{SchemaVersion: 1, Items: []*agentv1.MissionItem{{
		Sequence: 0, Frame: 0, Command: 16, Autocontinue: true,
		LatitudeE7: -353632620, LongitudeE7: 1491652370, AltitudeM: 20.1,
	}}}
}
