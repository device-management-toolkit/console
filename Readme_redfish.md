# Redfish Power Management PoC

This project demonstrates a minimal Redfish-compliant REST API for power management of systems managed by Intel AMT, using WS-Man under the hood. It integrates with the existing console service and the go-wsman-messages library.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)
- [Examples](#examples)


## Overview

This poc provides a Redfish API layer for Intel AMT power management operations. Built on top of the existing console service infrastructure, it translates standard Redfish REST calls into WS-Management protocol communications with Intel AMT devices.

## Architecture

### System Architecture

```mermaid
graph TB
    A[Redfish Client] -->|REST/JSON| B[Redfish API Layer]
    B -->|Go API Calls| C[Console Service]
    C -->|WS-Man Protocol| D[go-wsman-messages]
    D -->|XML/SOAP| E[Intel AMT Device]
    
    style A fill:#e1f5fe
    style B fill:#f3e5f5
    style C fill:#e8f5e8
    style D fill:#fff3e0
    style E fill:#fce4ec
```

### Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| **Redfish Client** | Standard Redfish REST API consumer (curl, Postman, management tools) |
| **Redfish API Layer** | Gin-based REST endpoints implementing Redfish specification |
| **Console Service** | Business logic layer for device management operations |
| **go-wsman-messages** | WS-Management protocol library for XML/SOAP communication |
| **Intel AMT Device** | Target managed system endpoint |

### Sequence Flow

```mermaid
sequenceDiagram
    participant RC as Redfish Client
    participant API as Redfish API Layer
    participant CS as Console Service
    participant WSM as go-wsman-messages
    participant AMT as Intel AMT Device
    
    RC->>+API: POST /redfish/v1/Systems/{id}/Actions/ComputerSystem.Reset
    Note over API: Parse ResetType & validate request
    
    API->>+CS: devices.Feature.SendPowerAction(id, action)
    Note over CS: Map action to WS-Man operation
    
    CS->>+WSM: Build WS-Man XML/SOAP message
    WSM->>+AMT: Send WS-Man request over HTTP
    
    AMT-->>-WSM: WS-Man response
    WSM-->>-CS: Parsed response data
    CS-->>-API: Operation result
    
    Note over API: Format Redfish JSON response
    API-->>-RC: HTTP 200 + Redfish JSON
```

## Technical Protocol Translation Specification

### Translation Location and Implementation Details

```mermaid
graph TD
    A[Redfish JSON Request] -->|HTTP POST| B[postSystemResetHandler]
    B -->|c.ShouldBindJSON| C[JSON Parsing Stage]
    C -->|body.ResetType| D[String Validation]
    D -->|Switch Statement| E[Action Constant Mapping]
    E -->|Integer Action| F[d.SendPowerAction Call]
    F -->|Console Service| G[WS-Management Layer]
    
    style B fill:#f9f,stroke:#333,stroke-width:2px
    style E fill:#bbf,stroke:#333,stroke-width:2px
    style F fill:#bfb,stroke:#333,stroke-width:2px
```

### Exact Translation Implementation

**File Location**: `internal/controller/http/redfish/system.go`  
**Function**: `postSystemResetHandler()`

#### Stage 1: JSON Deserialization (Lines 95-101)

```go

// Location: system.go
var body struct {
    ResetType string `json:"ResetType"`
}
if err := c.ShouldBindJSON(&body); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
}
```

- **Input**: Raw HTTP JSON payload `{"ResetType": "ForceOff"}`
- **Output**: Go struct with `body.ResetType = "ForceOff"`
- **Technology**: Gin framework's JSON binding

#### Stage 2: Redfish-to-Action Translation
```go

// Location: system.go
var action int

switch body.ResetType {
case resetTypeOn:           // "On" -> 2
    action = actionPowerUp
case resetTypeForceOff:     // "ForceOff" -> 8  
    action = actionPowerDown
case resetTypeForceRestart: // "ForceRestart" -> 10
    action = actionReset
case resetTypePowerCycle:   // "PowerCycle" -> 5
    action = actionPowerCycle
default:
    c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported ResetType"})
    return
}
```

#### Stage 3: Console Service Delegation

```go

// Location: system.go:119-125
res, err := d.SendPowerAction(c.Request.Context(), id, action)
if err != nil {
    l.Error(err, "http - redfish - ComputerSystem.Reset")
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
}

c.JSON(http.StatusOK, res)
```

### Translation Constants Definition

**File Location**: `internal/controller/http/redfish/system.go`  

```go
// Redfish ResetType constants
const (
    resetTypeOn           = "On"
    resetTypeForceOff     = "ForceOff"
    resetTypeForceRestart = "ForceRestart"
    resetTypePowerCycle   = "PowerCycle"
    
    // WS-Management action constants
    actionPowerUp    = 2
    actionPowerCycle = 5
    actionPowerDown  = 8
    actionReset      = 10
)
```

### Technical Translation Matrix

| Redfish JSON | Go Constant | Integer Value | WS-Man PowerState | AMT Action |
|-------------|-------------|---------------|-------------------|------------|
| `"On"` | `resetTypeOn` → `actionPowerUp` | `2` | CIM_PowerManagementService.RequestPowerStateChange(2) | Power On |
| `"ForceOff"` | `resetTypeForceOff` → `actionPowerDown` | `8` | CIM_PowerManagementService.RequestPowerStateChange(8) | Hard Power Off |
| `"ForceRestart"` | `resetTypeForceRestart` → `actionReset` | `10` | CIM_PowerManagementService.RequestPowerStateChange(10) | Reset |
| `"PowerCycle"` | `resetTypePowerCycle` → `actionPowerCycle` | `5` | CIM_PowerManagementService.RequestPowerStateChange(5) | Power Cycle |

### Interface Contract

**Method Signature**: 

```go

type Feature interface {
    SendPowerAction(ctx context.Context, guid string, action int) (any, error)
}
```

**Call Site**: `system.go`

```go

res, err := d.SendPowerAction(c.Request.Context(), id, action)
```

### Why No Separate Translator Module is Required

1. **Single Responsibility**: The `postSystemResetHandler` function in `system.go` handles the complete Redfish-to-WS-Management translation in 22 lines of code.

2. **Direct Mapping**: Translation is a simple 1:1 constant mapping requiring no complex logic or state management.

3. **Type Safety**: Go's type system ensures compile-time validation of the translation constants.

4. **Performance**: Direct switch statement provides O(1) lookup performance.

5. **Maintainability**: All translation logic is co-located with the API endpoint that uses it.

6. **Existing Abstraction**: The `devices.Feature.SendPowerAction()` interface already abstracts WS-Management protocol details.

---

## Getting Started

---

## Key Endpoints

- `GET /redfish/v1/`  
  Returns the Redfish Service Root.

- `GET /redfish/v1/Systems`  
  Returns a collection of managed systems.

- `GET /redfish/v1/Systems/{id}`  
  Returns details for a specific system, including its power state.

- `POST /redfish/v1/Systems/{id}/Actions/ComputerSystem.Reset`  
  Changes the power state of the specified system.

---

## How the Library Handles Protocol Translation

- The **Redfish API Layer** receives REST requests and parses the JSON payload.
- It maps Redfish actions (like `"ForceOff"`) to internal enums or constants.
- The **Console Service** translates these actions into WS-Man protocol operations, calling the appropriate methods in the `go-wsman-messages` library.
- The **go-wsman-messages** library constructs WS-Man-compliant XML/SOAP messages and parses responses.
- All WS-Man protocol details (namespaces, selectors, SOAP envelopes) are encapsulated in the library, so the API and Console layers remain clean and focused on business logic.
- The **Intel AMT Device** executes the WS-Man command and returns the result, which is translated back up the stack to a Redfish-compliant JSON response.

---

### Prerequisites

- Go 1.21 or later
- Access to Intel AMT-enabled devices
- Existing console service infrastructure

### Installation

Follow the steps as README.md

## Running the Service

```bash
go run ./cmd/app/main.go --config "./config/config.yml"
```

## API Reference

### Systems Collection

```http
GET /redfish/v1/Systems
```

Returns a collection of all managed systems.

### System Instance

```http
GET /redfish/v1/Systems/{systemId}
```

Returns detailed information about a specific system including power state.

### Power Management

```http
POST /redfish/v1/Systems/{systemId}/Actions/ComputerSystem.Reset
Content-Type: application/json

{
  "ResetType": "ForceOff"
}
```

Executes power management operations on the target system.

## Power State Mapping

| Redfish ResetType | Console Action | WS-Man PowerState | Description |
|------------------|---------------|-------------------|-------------|
| `On` | Power Up | `2` | Power on the system |
| `ForceOff` | Power Down | `8` | Immediate power off |
| `ForceRestart` | Reset | `10` | Hard reset |
| `PowerCycle` | Power Cycle | `5` | Power cycle operation |
| `GracefulShutdown` | Soft Power Off | `12` | Graceful shutdown |

## Examples

## Authorization

### Authorization Command Request
```bash
curl -s -X POST http://localhost:8181/api/v1/authorize   -H "Content-Type: application/json"   -d "{\"username\":\"$API_USERNAME\",\"password\":\"$API_PASSWORD\"}"
```

### Authorization Command Response
```json
{
    "token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTk1MTA3MjV9.2T1EZcyBhSJUfIT-ACnzTIBaU1AsXhLnCEek2GIS94"
}
```

## Get Session Service

### Get Session Command Request
```bash
curl -X GET http://localhost:8181/api/redfish/v1/SessionServic -H "Authorization: Bearer $AUTH_TOKEN"
```

### Get Session Command Response
```json
{
    "@odata.id":"/redfish/v1/SessionService",
    "@odata.type":"#SessionService.v1_0_0.SessionService",
    "Id":"SessionService",
    "Name":"Redfish Session Service",
    "ServiceEnabled":true,
    "SessionTimeout":30,
    "Sessions":
    {
        "@odata.id":"/redfish/v1/SessionService/Sessions"
    }
}
```

## Get Metadata

### Get Metadata Command Request
```bash
curl -X GET http://localhost:8181/api/redfish/v1/$metadata -H "Authorization: Bearer $AUTH_TOKEN"
```

### Get Metadata Command Response
```json
{
    "@odata.id": "/redfish/v1/",
    "@odata.type": "#ServiceRoot.v1_0_0.ServiceRoot",
    "Id": "RootService",
    "Links": {
        "Sessions": {
            "@odata.id": "/redfish/v1/SessionService/Sessions"
        }
    },
    "Name": "Redfish Service Root",
    "RedfishVersion": "1.0.0",
    "SessionService": {
        "@odata.id": "/redfish/v1/SessionService"
    },
    "Systems": {
        "@odata.id": "/redfish/v1/Systems"
    }
} 
```

## Get System Details

### Get System Details Command Request
 ```bash
curl -X GET http://localhost:8181/api/redfish/v1/Systems -H "Authorization: Bearer $AUTH_TOKEN"
```

### Get System Details Command Response
```json
{
    "@odata.id": "/redfish/v1/Systems",
    "@odata.type": "#ComputerSystemCollection.ComputerSystemCollection",
    "Members": [
        {
            "@odata.id": "/redfish/v1/Systems/0313b93f-2a7c-41e8-b603-29c1602dcef9"
        }
    ],
    "Members@odata.count": 1,
    "Name": "Computer System Collection"
}
```


## Get System details by ID

### Get System Details by ID Command Request
```bash
curl -X GET http://localhost:8181/api/redfish/v1/Systems/0313b93f-2a7c-41e8-b603-29c1602dcef9   -H "Authorization: Bearer $AUTH_TOKEN"
```

### Get System Details by ID SUCCESS Command Response
```json
{
    "@odata.id": "/redfish/v1/Systems/0313b93f-2a7c-41e8-b603-29c1602dcef9",
    "@odata.type": "#ComputerSystem.v1_0_0.ComputerSystem",
    "Actions": {
        "#ComputerSystem.Reset": {
            "ResetType@Redfish.AllowableValues": [
                "On",
                "ForceOff",
                "ForceRestart",
                "PowerCycle"
            ],
            "target": "/redfish/v1/Systems/0313b93f-2a7c-41e8-b603-29c1602dcef9/Actions/ComputerSystem.Reset"
        }
    },
    "Id": "0313b93f-2a7c-41e8-b603-29c1602dcef9",
    "Name": "Computer System 0313b93f-2a7c-41e8-b603-29c1602dcef9",
    "PowerState": "On"
}
```

### Get System Details by ID ERROR Command Response

### Basic Power Operations

#### Power OFF System

##### POwer OFF System Command Request
```bash
curl -X POST http://localhost:8181/api/redfish/v1/Systems/0313b93f-2a7c-41e8-b603-29c1602dcef9/Actions/ComputerSystem.Reset   -H "Authorization: Bearer $AUTH_TOKEN"   -H "Content-Type: application/json"   -d '{"ResetType":"ForceOff"}'
```

##### Power OFF System SUCCESS Command Response
```json
{"ReturnValue":0}
```

##### Power OFF System ERROR Command Response (Device ID Does Not Exist)
```json
{"error":"DevicesUseCase -  - : "}
```

##### Power ON System Command Request
```bash
curl -X POST http://localhost:8181/api/redfish/v1/Systems/0313b93f-2a7c-41e8-b603-29c1602dcef9/Actions/ComputerSystem.Reset   -H "Authorization: Bearer $AUTH_TOKEN"   -H "Content-Type: application/json"   -d '{"ResetType":"On"}'
```

##### Power ON System SUCCESS Command Response
```json
{"ReturnValue":0}
```

##### Power ON System ERROR Command Response (Device ID Does Not Exist)
```json
{"error":"DevicesUseCase -  - : "}
```