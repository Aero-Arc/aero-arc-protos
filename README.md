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

The first contract slice carries a schema-versioned `MissionPlan` containing at
most 200 canonical `MissionItem` records. For schema version 1, sequence numbers
are contiguous from zero, coordinates use signed degrees times 1e7, and unknown
fields are rejected. Canonical items exclude autopilot HOME entries and
Mission Planner/QGC export metadata. `MissionItem.current` is reserved and must
be false because ArduPilot changes it dynamically during execution and readback.
Parameters 1–3 use positive zero. Parameter 4 uses positive zero for
`MAV_CMD_NAV_WAYPOINT` (16) and `MAV_CMD_NAV_TAKEOFF` (22), but exactly `+1` for
`MAV_CMD_NAV_LAND` (21), matching ArduPilot's stable stored/readback form.
The mission digest is lowercase hexadecimal SHA-256 of the deterministic
protobuf serialization of this normalized `MissionPlan`.

`DeployMission` targets only the Agent session selected by the Relay at command
admission. The command ID and payload remain stable across reconciliation.
`APPLIED` and `ALREADY_APPLIED` mean the Agent read back the onboard mission and
confirmed its digest. `OUTCOME_UNKNOWN` is not permission to issue a new command
ID: the caller reconciles the same logical command. Binding and onboard digest
mismatches are explicit terminal outcomes.

### Optional Tooling (Future)

You can add:

- `buf.yaml` and `buf.gen.yaml` in the repo root if you choose [Buf](https://buf.build/) for linting and codegen.
- Language-specific generation scripts under `tools/` (e.g., `tools/gen-go.sh`, `tools/gen-ts.sh`).
