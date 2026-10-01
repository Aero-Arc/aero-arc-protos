# Completion receipt canonical encoding, version 1

The receipt `payload_sha256` is lowercase hexadecimal SHA-256 of the following
canonical bytes. A runtime's deterministic protobuf serialization is not the
specification. Consumers decode the received message, validate it, and reconstruct
these bytes before comparing receipts.

Use protobuf wire tags `(field_number << 3) | wire_type`, encoded as minimal
unsigned base-128 varints. Emit each field exactly once in ascending field-number
order. Strings are their exact valid UTF-8 bytes (no Unicode normalization),
prefixed by a minimal varint byte length, with wire type 2. Positive integer
values use minimal unsigned varints, wire type 0; do not use ZigZag. Omit empty
strings and zero scalar values. Validation currently requires all fields below
to be nonzero/nonempty. Unknown fields at either message level are rejected;
future extensions must explicitly version the canonical contract.

The outer `FlightCompletionEvidence` fields are:

| Number | Value | Wire type |
|---|---|---|
|1|event_id|2|
|2|agent_id|2|
|3|canonical OperationContext bytes, length-prefixed|2|
|4|mission_id|2|
|5|mission_digest|2|
|6|start_command_id|2|
|7|outcome|2|
|8|airborne_at_unix_ns|0|
|9|terminal_at_unix_ns|0|
|10|landed_at_unix_ns|0|
|11|disarmed_at_unix_ns|0|
|12|observation_epoch|2|

Nested OperationContext uses the same rules: flight_id (1, string), intent_id
(2, string), intent_version (3, uint32), aircraft_id (4, string).

The literal golden vector in `evidence_test.go` is independently encoded and
covers multi-byte integer varints and nested length prefixes. These bytes preserve receipts
already produced by the Go implementation for known-field, valid evidence.
The separate runtime comparison test protects that compatibility; it does not
define canonical encoding.
