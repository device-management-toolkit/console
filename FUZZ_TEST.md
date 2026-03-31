# Fuzz Testing

## Overview

Console employs two complementary fuzzing strategies:

1. **Internal function fuzzing** — Go's built-in `go test -fuzz` framework targets parsing, transformation, validation, and cryptographic functions at the unit level.
2. **API fuzzing** — *(planned)* HTTP-level fuzzing of Console's REST API endpoints.

---

## Internal Function Fuzzing

Uses Go's native fuzzing to test internal functions for crash resistance, determinism, and input-safety. There are **17 fuzz targets** across **11 test files** covering four categories: JSON/DTO validation, use-case DTO↔entity transforms, cryptographic parsing, and string parsing.

### Coverage Summary

#### 1. JSON Deserialization & DTO Validation (7 targets)

All targets live in a single file and share a generic helper `fuzzJSONAndValidate[T]` that verifies `json.Unmarshal` determinism, `reflect.DeepEqual` consistency, and struct-level validator stability.

| Target | DTO Type | Validators Exercised | File |
|--------|----------|----------------------|------|
| `FuzzDeviceJSONProcessing` | `Device` | default | `internal/entity/dto/v1/json_fuzz_test.go` |
| `FuzzProfileJSONProcessing` | `Profile` | `ValidateAMTPassOrGenRan`, `ValidateCIRAOrTLS`, `ValidateWiFiDHCP` | `internal/entity/dto/v1/json_fuzz_test.go` |
| `FuzzDomainJSONProcessing` | `Domain` | `ValidateAlphaNumHyphenUnderscore` | `internal/entity/dto/v1/json_fuzz_test.go` |
| `FuzzCIRAConfigJSONProcessing` | `CIRAConfig` | default | `internal/entity/dto/v1/json_fuzz_test.go` |
| `FuzzWirelessConfigJSONProcessing` | `WirelessConfig` | `ValidateAuthandIEEE` | `internal/entity/dto/v1/json_fuzz_test.go` |
| `FuzzIEEE8021xJSONProcessing` | `IEEE8021xConfig` | `AuthProtocolValidator` | `internal/entity/dto/v1/json_fuzz_test.go` |
| `FuzzProfileWiFiJSONProcessing` | `ProfileWiFiConfigs` | default | `internal/entity/dto/v1/json_fuzz_test.go` |

**What is tested:** arbitrary JSON payloads (valid, malformed, deeply nested, oversized, unicode/null-byte) are deserialized twice and validated twice. Failures in determinism, panics, or mismatched validation results are caught.

#### 2. Use-Case DTO↔Entity Transform Round-Trips (7 targets)

Each target fuzzes both `dtoToEntity` and `entityToDTO` in its respective use-case package. Cryptographic dependencies are satisfied with mock implementations. All targets assert determinism via dual invocation and `reflect.DeepEqual`.

| Target | Package | Functions Under Test | File |
|--------|---------|----------------------|------|
| `FuzzCIRAConfigTransforms` | `ciraconfigs` | `dtoToEntity`, `entityToDTO` | `internal/usecase/ciraconfigs/transform_fuzz_test.go` |
| `FuzzDeviceTransforms` | `devices` | `dtoToEntity`, `entityToDTO` + GUID lowercasing, cert-hash nil check | `internal/usecase/devices/transform_fuzz_test.go` |
| `FuzzDomainTransforms` | `domains` | `dtoToEntity`, `entityToDTO` + RFC3339 expiration parsing, password encryption | `internal/usecase/domains/transform_fuzz_test.go` |
| `FuzzIEEE8021xConfigTransforms` | `ieee8021xconfigs` | `dtoToEntity`, `entityToDTO` | `internal/usecase/ieee8021xconfigs/transform_fuzz_test.go` |
| `FuzzProfileTransforms` | `profiles` | `dtoToEntity`, `entityToDTO` + tag join consistency | `internal/usecase/profiles/transform_fuzz_test.go` |
| `FuzzProfileWiFiConfigTransforms` | `profilewificonfigs` | `dtoToEntity`, `entityToDTO` | `internal/usecase/profilewificonfigs/transform_fuzz_test.go` |
| `FuzzWirelessConfigTransforms` | `wificonfigs` | `dtoToEntity`, `entityToDTO` + link-policy nil handling | `internal/usecase/wificonfigs/transform_fuzz_test.go` |

**What is tested:** fuzzed field values (strings, ints, bools, timestamps, oversized/unicode/null-byte data) are passed through transform functions. Panics, non-deterministic results, and invariant violations (e.g. GUID not lowercased, nil pointer where expected) are caught.

#### 3. Cryptographic Parsing (2 targets)

| Target | Package | Function Under Test | File |
|--------|---------|---------------------|------|
| `FuzzParseCertificateFromPEM` | `certificates` | `ParseCertificateFromPEM` | `internal/certificates/generate_fuzz_test.go` |
| `FuzzDecryptAndCheckCertExpiration` | `domains` | `DecryptAndCheckCertExpiration` | `internal/usecase/domains/usecase_fuzz_test.go` |

**What is tested:**
- `FuzzParseCertificateFromPEM`: valid, truncated, corrupted, swapped, and random PEM cert+key pairs. Asserts determinism, no data-alongside-error, no nil-without-error, and serial/key consistency.
- `FuzzDecryptAndCheckCertExpiration`: valid, expired, corrupted, and random base64-encoded PKCS12 blobs with varied passwords. Asserts determinism, no expired-cert-without-error, and byte-level certificate consistency.

#### 4. String Parsing (1 target)

| Target | Package | Function Under Test | File |
|--------|---------|---------------------|------|
| `FuzzParseInterval` | `devices` | `ParseInterval` (ISO 8601 duration → minutes) | `internal/usecase/devices/alarms_fuzz_test.go` |

**What is tested:** arbitrary strings (empty, malformed, oversized, unicode/control-character, and valid ISO 8601 durations) are parsed twice. Asserts determinism, no panics, and consistent error/result pairs across invocations.

---

### Seed Corpus Strategy

All targets use explicit `f.Add()` seeds covering:
- **Happy path:** well-formed, realistic inputs.
- **Empty/zero:** empty strings, zero ints, nil-equivalent bools.
- **Unicode & control characters:** CJK, emoji, null bytes (`\x00`), newlines.
- **Oversized inputs:** strings up to 4 KB, arrays with 4096 elements.
- **Type confusion:** wrong JSON types (number where string expected, etc.).
- **Boundary values:** `int` min/max, year-zero and year-9999 timestamps, port 65535.
- **Crypto edge cases:** truncated PEM, corrupted base64, swapped cert/key, wrong passwords, expired certificates.

---

### Running Internal Fuzz Tests

#### Prerequisites

```sh
cp .env.example .env   # Makefile includes .env
```

#### Make Targets

```sh
# List all 17 fuzz targets
make fuzz-list

# Run a single target (recommended for local dev)
make fuzz-one PKG=./internal/usecase/devices TARGET=FuzzParseInterval FUZZTIME=30s

# Quick smoke: run every target once (seed corpus only)
make fuzz-smoke

# Full run: every target sequentially with time budget
make fuzz-all FUZZTIME=2m
```

#### Direct go test (single target)

```sh
go test ./internal/usecase/devices -run='^$' -fuzz='^FuzzParseInterval$' -fuzztime=30s
```

> **Note:** Go requires `-fuzz` to match exactly one fuzz function per package. Packages with multiple targets (e.g. `internal/entity/dto/v1` has 7) must be run one target at a time. The `make fuzz-all` target handles this automatically.

#### CI Usage

| Trigger | Command | Purpose |
|---------|---------|---------|
| Pull request | `make fuzz-smoke` | Replay seed corpus, catch regressions |
| Nightly/weekly schedule | `make fuzz-all FUZZTIME=2m` | Discover new crashes with mutation |

---

### File Index

| File | Targets | Package |
|------|---------|---------|
| `internal/certificates/generate_fuzz_test.go` | 1 | `certificates` |
| `internal/entity/dto/v1/json_fuzz_test.go` | 7 | `dto` |
| `internal/usecase/ciraconfigs/transform_fuzz_test.go` | 1 | `ciraconfigs` |
| `internal/usecase/devices/alarms_fuzz_test.go` | 1 | `devices` |
| `internal/usecase/devices/transform_fuzz_test.go` | 1 | `devices` |
| `internal/usecase/domains/transform_fuzz_test.go` | 1 | `domains` |
| `internal/usecase/domains/usecase_fuzz_test.go` | 1 | `domains` |
| `internal/usecase/ieee8021xconfigs/transform_fuzz_test.go` | 1 | `ieee8021xconfigs` |
| `internal/usecase/profiles/transform_fuzz_test.go` | 1 | `profiles` |
| `internal/usecase/profilewificonfigs/transform_fuzz_test.go` | 1 | `profilewificonfigs` |
| `internal/usecase/wificonfigs/transform_fuzz_test.go` | 1 | `wificonfigs` |
| **Total** | **17** | **9 packages** |

---

## API Fuzzing

*Planned — this section will document HTTP-level fuzz testing of Console's REST API endpoints.*
