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

### Optional Tooling (Future)

You can add:

- `buf.yaml` and `buf.gen.yaml` in the repo root if you choose [Buf](https://buf.build/) for linting and codegen.
- Language-specific generation scripts under `tools/` (e.g., `tools/gen-go.sh`, `tools/gen-ts.sh`).



