# Canonical command digest v1

`Digest` returns lowercase hexadecimal SHA-256 of the following byte sequence.
It is independent of protobuf serialization and JSON formatting. Validation fails
for unknown protobuf fields, invalid authority/expiry, nonfinite floats, negative
zero, unsupported capability/recovery combinations, or inconsistent bindings.

Prefix: UTF-8 `aeroarc-command-v1` followed by one zero byte.
All integers are big endian. Strings are a uint32 UTF-8 byte length followed by
those bytes. No Unicode normalization is performed. Signed integers use two's
complement. Booleans occupy one byte (0 or 1). Float32s use IEEE-754 bits encoded
as uint32. Arrays begin with a uint32 element count.

Fields in exact order:

1. operator, aircraft, Agent, flight, intent (strings)
2. intent version (uint32)
3. definition (string), definition version (uint32), capability (string)
4. issued and expires (int64 Unix milliseconds)
5. recovery policy (string)
6. MAVLink command (uint32), parameter bits (uint32 array)
7. command-int flag (bool), frame (uint32), x/y (int32), z bits (uint32)
8. vehicle profile and observation predicate (strings), expected custom mode (uint32)
9. mission digest, mission ID, deployment ID, mission command ID (strings)
10. mission version (uint32)

Fields unused by the chosen execution variant are zero or empty (including an
empty parameter array for mission upload). Mission content uses the existing
`missiondigest` contract. For a MAVLink mission precondition, deployment and
mission-command IDs are empty. Command UUID and command_digest are excluded;
Agent target is included while Relay placement and attempt identity are excluded.
The golden vector in digest_test.go fixes cross-version byte compatibility.
Changes to this encoding require a new canonical version, never silently altering
v1. New approved definition versions may use the existing execution capabilities.
