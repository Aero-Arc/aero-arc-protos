# Aero Arc protobuf contributor context

- `proto/` is the source of truth; generated Go clients are checked in under
  `gen/go/` and must not be edited by hand.
- Run `buf generate`, `buf lint`, `go test ./...`, and `go vet ./...` after a
  contract change. Prefer `buf generate --path proto/<path>/<file>.proto` for a
  focused schema change.
- Preserve protobuf compatibility: never reuse or renumber a published field
  number. Reserve removed field names and numbers.
- Registry agent placement is relay-owned. `RegisterAgentRequest.relay_id` sets
  ownership and `HeartbeatAgentRequest.relay_id` proves the sending relay still
  owns that placement.
- Exported handwritten Go helpers require identifier-led Go doc comments that
  explain behavior, parameters, return values, and compatibility-sensitive
  errors. Generated Go documentation belongs to the protobuf toolchain; improve
  source comments in `proto/` and regenerate instead of editing `gen/go/`.
- Commits require a Developer Certificate of Origin sign-off (`git commit -s`).
