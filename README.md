## aero-arc-protos

Central repository for all Aero Arc Protocol Buffers (`.proto` files) used by different clients and services.

### Goals

- **Single source of truth** for message and service definitions
- **Shared common types** reused across clients
- **Clear separation** between public/common contracts and client-specific APIs

### Directory Layout

Top-level layout:

- `proto/` – All `.proto` definitions, organized by domain and client
- `tools/` – Optional helper scripts or codegen configurations (e.g., Buf, `protoc` wrappers)

Proto layout:

- `proto/aeroarc/common/v1/` – Shared, client-agnostic messages and services
- `proto/aeroarc/platform/v1/` – Core platform APIs used by multiple clients
- `proto/aeroarc/clients/<client_name>/v1/` – Client-specific APIs

Example:

```text
proto/
  aeroarc/
    common/
      v1/
        types.proto          # Generic/common messages and enums
        auth.proto           # Authentication-related messages
    platform/
      v1/
        telemetry.proto      # Example: telemetry/metrics contracts
        control.proto        # Example: control/command APIs
    clients/
      web/
        v1/
          web_api.proto      # Web client-specific RPCs
      mobile/
        v1/
          mobile_api.proto   # Mobile client-specific RPCs
      embedded/
        v1/
          embedded_api.proto # Embedded/edge client-specific RPCs
```

### Package Naming Convention

- Use **lowercase**, dot-separated packages matching the directory layout.
- Example packages:
  - `package aeroarc.common.v1;`
  - `package aeroarc.platform.v1;`
  - `package aeroarc.clients.web.v1;`

This keeps imports and generated code consistent and easy to navigate.

### Adding New Protos

1. **Choose the right area**:
   - Shared or reusable types → `proto/aeroarc/common/v1/`
   - Cross-client platform APIs → `proto/aeroarc/platform/v1/`
   - Client-specific APIs → `proto/aeroarc/clients/<client_name>/v1/`
2. **Create a new `.proto` file** following the package naming convention.
3. **Reuse common messages** from `aeroarc.common.v1` where possible.

### Telemetry Cursor Compatibility

`TelemetryFrame` uses `(wal_id, seq)` as its durable transport cursor.
`wal_id` identifies one Agent WAL append generation and `seq` is monotonic only
within that generation. An Agent rotates the generation whenever it opens a WAL
for new capture, while a frame already persisted retains its original pair
through retry and restart.

The field is additive on the wire, but enforcement is a deployment concern:
publish Protos first, deploy producers that populate `wal_id`, and only then
deploy consumers that reject a missing or invalid value. A consumer must not
silently change an existing storage identity formula merely because the new
cursor is available; that requires an explicit schema-version transition.

### Conformance Assignment Lifecycle

The public conformance control plane uses separate, idempotent assignment
commands. `PrepareAssignment` persists a candidate, `ArmAssignment` marks that
candidate ready after validation, and `CutoverAssignment` establishes its
half-open telemetry-authority interval at an explicit event-time boundary.
Arming alone never authorizes telemetry. A candidate that will not be cut over
can be removed with `CancelAssignmentCandidate`.

### Mission Deployment Contract

Mission deployment is strictly separate from operational-intent geometry.
`MissionBinding` ties one immutable mission version and digest to a deployment,
operator, aircraft, flight, and exact intent version; it never authorizes or
modifies an operational volume.

The first contract slice carries a schema-versioned `MissionPlan` containing 1
to 200 canonical `MissionItem` records. Schema version 1 allows only
`MAV_FRAME_GLOBAL` (0), `MAV_CMD_NAV_WAYPOINT` (16), `MAV_CMD_NAV_LAND` (21),
and `MAV_CMD_NAV_TAKEOFF` (22). Sequence numbers are contiguous from zero,
`autocontinue` is true, coordinates use signed degrees times 1e7, and unknown
fields are rejected. Canonical items exclude autopilot HOME entries and Mission
Planner/QGC export metadata. `MissionItem.current` is reserved and must be false
because ArduPilot changes it dynamically during execution and readback.
Readback reconstruction must discard ArduPilot's synthetic HOME item at wire
sequence zero, reassign the remaining operational items contiguous canonical
sequences from zero, discard the dynamic current marker, and force `current` to
false before validation or canonical digest calculation.
Parameters 1–3 use positive zero. Parameter 4 uses positive zero for waypoint
and takeoff, but exactly `+1` for land, matching ArduPilot's stable
stored/readback form. Altitude is encoded as protobuf `float` so the canonical
digest contains the exact float32 value transported by MAVLink; producers must
also require that it round-trip through ArduPilot's signed centimeter storage.
That conversion multiplies the float32 altitude by float32 `100`, requires the
float32 product to satisfy `-2147483648 <= product < 2147483648` before any
integer conversion, converts it to signed int32 by truncating toward zero, and
multiplies the stored value by float32 `0.01` for readback; the readback float32
bits must equal the input bits. Rounding to the nearest centimeter is not
equivalent (for example, float32 `16.8` stores as `1679` cm and reads back as
`16.79`, so it is rejected).
Coordinates are exact `MISSION_ITEM_INT` E7 values and do not need to round-trip
through legacy `MISSION_ITEM` float coordinates; an Agent unable to use the INT
exchange must fail closed rather than narrowing the canonical contract.

The schema-one digest encoding is independent of protobuf runtime behavior. It
starts with ASCII `aeroarc-mission-plan-v1` and a zero byte, followed by the item
count as unsigned 32-bit big-endian. Each item then encodes, in order: sequence,
frame, and command as unsigned 32-bit big-endian; current and autocontinue as
one byte each; parameters 1–4 as IEEE-754 binary64 bits in big-endian order;
latitude and longitude E7 as signed 32-bit two's-complement big-endian; and
altitude as IEEE-754 binary32 bits in big-endian order. Protobuf tags, lengths,
wire ordering, unknown fields, and alignment padding are not included. The
mission digest is lowercase hexadecimal SHA-256 of exactly these bytes after
validation. Go producers and consumers should use the shared `missiondigest`
package rather than duplicating the encoder; other runtimes must match the same
byte specification and golden vector.

`DeployMission` targets only the Agent session selected by the Relay at command
admission. The Agent first validates and canonically digests the supplied plan,
requires that digest to equal `MissionBinding.mission_digest`, and rejects a
mismatch before any onboard readback or upload. This supplied-plan check is
separate from later comparing onboard readback with the binding digest. Before
admitting a first-seen command or reporting its first success,
including `ALREADY_APPLIED`, the Agent requires an active operation context
whose aircraft, flight, intent, and intent version exactly equal the mission
binding; otherwise it returns `BINDING_MISMATCH`. The command ID and payload
remain stable across reconciliation. After validating a first-seen unexpired
command and its active binding, but before its first onboard readback, the Agent
durably records the exact payload fingerprint as admitted with no effect
started. The Agent looks up that record and verifies its fingerprint before
checking expiry. Exact durable terminal replay and readback-only reconciliation
remain available after the active context changes because they create no new
effect. Expiry prevents first admission and new uploads, but a matching
previously admitted command still replays its terminal result or performs an
effect-free recovery readback. For an admitted command whose terminal result
was not stored, post-expiry reconciliation is readback-only whether or not an
upload started: a matching digest yields `ALREADY_APPLIED`; a mismatch yields
`ONBOARD_MISSION_MISMATCH` and never starts a replacement upload. Before expiry,
an admitted mismatch may be uploaded only after rechecking the exact active
binding and every safety fence. Before calling any transport operation
that can mutate the autopilot, the Agent durably commits the exact payload
fingerprint and an `effect_started` state; a failed write-ahead commit prevents
the upload and returns `TEMPORARY_ERROR`. Readback outcomes are stored before
response. If outcome persistence fails, the durable admission record and, when
applicable, the started-effect record remain recovery anchors so a retry cannot
appear first-seen. `APPLIED` and
`ALREADY_APPLIED` mean the Agent read back the onboard mission and confirmed its
digest. `uploaded_item_count` is the number transferred by the execution that
produced the result: `APPLIED` reports the complete plan length and a
readback-only `ALREADY_APPLIED` reports zero; durable replay preserves that
original count. `OUTCOME_UNKNOWN` is not permission to issue a new command ID: the
caller reconciles the same logical command. Binding and onboard digest
mismatches are explicit terminal outcomes.

### Optional Tooling (Future)

You can add:

- `buf.yaml` and `buf.gen.yaml` in the repo root if you choose [Buf](https://buf.build/) for linting and codegen.
- Language-specific generation scripts under `tools/` (e.g., `tools/gen-go.sh`, `tools/gen-ts.sh`).
