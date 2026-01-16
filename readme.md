---
name: Go Validator Monitor
overview: Build a high-performance Ethereum Validator Monitor in Go that polls the Beacon Node API to track validator status, effective balance, attestation duties, and consensus layer rewards using a worker pool pattern with exponential backoff. Persists all data to ScyllaDB with configurable TTL.
todos:
  - id: setup-module
    content: Create go.mod, go.sum with dependencies (zerolog, gocql, x/time/rate, yaml.v3)
    status: completed
  - id: config-loader
    content: Implement YAML config loader with ScyllaDB settings
    status: completed
  - id: backoff-util
    content: Implement exponential backoff utility in pkg/backoff/backoff.go
    status: completed
  - id: scylla-client
    content: Build ScyllaDB client with connection pooling and auto-migration
    status: completed
  - id: scylla-schema
    content: Define ScyllaDB schema for validator data tables
    status: completed
  - id: scylla-repository
    content: Implement repository layer for validator data persistence
    status: completed
  - id: beacon-client
    content: Build HTTP client with rate limiting and connection pooling
    status: completed
  - id: beacon-types
    content: Define Beacon API response structs with MaxEB support
    status: completed
  - id: beacon-endpoints
    content: Implement validators, duties, and rewards API methods
    status: completed
  - id: worker-pool
    content: Create worker pool pattern in internal/monitor/worker.go
    status: completed
  - id: scheduler
    content: Implement slot-based scheduler for epoch-aware polling
    status: completed
  - id: monitor-loop
    content: Build core monitoring orchestration with DB writes
    status: completed
  - id: main-entry
    content: Create main.go with graceful shutdown handling
    status: completed
  - id: example-config
    content: Create example config.yaml file with ScyllaDB settings
    status: completed
---

# Ethereum Validator Monitor in Go

## Architecture Overview

```mermaid
flowchart TB
    subgraph main [Main Process]
        Config[Config Loader]
        Scheduler[Slot Scheduler]
        Shutdown[Graceful Shutdown]
    end
    
    subgraph beacon [internal/beacon]
        Client[HTTP Client]
        RateLimiter[Rate Limiter]
        Backoff[Exponential Backoff]
    end
    
    subgraph monitor [internal/monitor]
        WorkerPool[Worker Pool]
        StatusTracker[Status Tracker]
        DutyFetcher[Duty Fetcher]
        RewardCalc[Reward Calculator]
    end
    
    subgraph storage [internal/storage]
        ScyllaClient[ScyllaDB Client]
        Repository[Validator Repository]
        Migration[Schema Migration]
    end
    
    subgraph output [Output]
        Logger[zerolog JSON Logger]
        ScyllaDB[(ScyllaDB)]
    end
    
    Config --> Scheduler
    Config --> ScyllaClient
    Scheduler --> WorkerPool
    WorkerPool --> Client
    Client --> RateLimiter
    RateLimiter --> Backoff
    StatusTracker --> Repository
    DutyFetcher --> Repository
    RewardCalc --> Repository
    Repository --> ScyllaDB
    StatusTracker --> Logger
    DutyFetcher --> Logger
    RewardCalc --> Logger
```

## Data Flow

```mermaid
sequenceDiagram
    participant S as Scheduler
    participant W as Worker Pool
    participant B as Beacon API
    participant R as Repository
    participant DB as ScyllaDB
    participant L as Logger
    
    S->>W: Dispatch slot job
    W->>B: GET /validators/{id}
    B-->>W: Status + Balances
    W->>R: SaveValidatorSnapshot
    R->>DB: INSERT with TTL
    DB-->>R: OK
    W->>L: Log JSON output
    
    Note over S,DB: On epoch boundary
    W->>B: GET /duties/attester/{epoch}
    B-->>W: Attestation duties
    W->>R: SaveAttestationDuties
    R->>DB: INSERT with TTL
    
    Note over S,DB: After epoch finalization
    W->>B: GET /rewards/attestations/{epoch}
    B-->>W: Rewards breakdown
    W->>R: SaveRewards
    R->>DB: INSERT with TTL
```

## Project Structure

```
pauli/
├── main.go                      # Entry point, graceful shutdown
├── go.mod
├── go.sum
├── config.yaml                  # Example configuration
├── internal/
│   ├── config/
│   │   └── config.go            # YAML config loader
│   ├── beacon/
│   │   ├── client.go            # HTTP client with connection pooling
│   │   ├── types.go             # API response structs
│   │   ├── validators.go        # Validator status endpoint
│   │   ├── duties.go            # Attestation duties endpoint
│   │   └── rewards.go           # Attestation rewards endpoint
│   ├── storage/
│   │   ├── scylla.go            # ScyllaDB client and connection
│   │   ├── migrations.go        # Schema auto-migration
│   │   ├── models.go            # Database models
│   │   └── repository.go        # Data access layer
│   └── monitor/
│       ├── monitor.go           # Core monitoring loop
│       ├── worker.go            # Worker pool implementation
│       └── scheduler.go         # Slot-based scheduling
└── pkg/
    └── backoff/
        └── backoff.go           # Exponential backoff utility
```

## Key Implementation Details

### 1. Configuration (YAML)

```yaml
beacon_node_url: "http://localhost:5052"
validators:
  - 12345
  - 67890
  - 111213
polling_interval_slots: 1        # Poll every slot (12s)
worker_pool_size: 10
rate_limit:
  requests_per_second: 50
  burst: 100
http:
  timeout_seconds: 30
  max_idle_conns: 100

# ScyllaDB Configuration
scylladb:
  hosts:
    - "127.0.0.1:9042"
  keyspace: "validator_monitor"
  replication_factor: 3
  consistency: "local_quorum"    # For sync writes
  timeout_seconds: 10
  max_retries: 3
  ttl_days: 90                   # Configurable data retention
```

### 2. High-Performance HTTP Client

- Use `net/http` with optimized `Transport` settings:
                                                                - `MaxIdleConns: 100`
                                                                - `MaxIdleConnsPerHost: 100`
                                                                - `IdleConnTimeout: 90s`
                                                                - HTTP/2 enabled by default
- Implement token bucket rate limiter using `golang.org/x/time/rate`
- Pre-allocate buffers for JSON decoding

### 3. Worker Pool Pattern

- Fixed pool of N goroutines (configurable, default 10)
- Job channel distributes validator indices to workers
- Result channel collects monitoring results
- Prevents goroutine explosion when monitoring 100+ validators
```mermaid
flowchart LR
    JobQueue[Job Channel] --> W1[Worker 1]
    JobQueue --> W2[Worker 2]
    JobQueue --> W3[Worker N]
    W1 --> Results[Result Channel]
    W2 --> Results
    W3 --> Results
    Results --> Logger[JSON Logger]
```


### 4. Exponential Backoff

Handle `429` and `503` errors with:

- Initial delay: 100ms
- Max delay: 30s
- Multiplier: 2x
- Jitter: +/- 20%

### 5. Monitor Loop (per epoch/slot)

1. **On each slot**: Fetch validator status and effective balance
2. **On epoch boundary**: Fetch attestation duties for next epoch
3. **After epoch finalization**: Fetch attestation rewards for previous epoch

### 6. JSON Log Output Format

```json
{
  "level": "info",
  "time": "2026-01-16T10:00:00Z",
  "slot": 1234567,
  "validator_index": 12345,
  "status": "active_ongoing",
  "effective_balance_gwei": 64000000000,
  "duty_success": true,
  "reward_gwei": 12500
}
```

### 7. MaxEB (EIP-7251) Support

- Parse `effective_balance` as `uint64` (supports up to 2048 ETH = 2048e9 Gwei)
- No hardcoded 32 ETH assumptions in balance validation

### 8. ScyllaDB Schema

Four tables optimized for time-series queries with partition by validator and clustering by slot/epoch:

**Table: validator_snapshots** (per-slot balance and status)

```cql
CREATE TABLE IF NOT EXISTS validator_snapshots (
    validator_index BIGINT,
    slot            BIGINT,
    status          TEXT,
    balance         BIGINT,          -- Actual balance in Gwei
    effective_balance BIGINT,        -- Effective balance in Gwei (MaxEB aware)
    timestamp       TIMESTAMP,
    PRIMARY KEY ((validator_index), slot)
) WITH CLUSTERING ORDER BY (slot DESC)
  AND default_time_to_live = 7776000;  -- 90 days, configurable
```

**Table: attestation_duties** (per-epoch duty assignments)

```cql
CREATE TABLE IF NOT EXISTS attestation_duties (
    validator_index   BIGINT,
    epoch             BIGINT,
    slot              BIGINT,
    committee_index   INT,
    committee_position INT,
    timestamp         TIMESTAMP,
    PRIMARY KEY ((validator_index), epoch, slot)
) WITH CLUSTERING ORDER BY (epoch DESC, slot DESC)
  AND default_time_to_live = 7776000;
```

**Table: attestation_rewards** (per-epoch rewards breakdown)

```cql
CREATE TABLE IF NOT EXISTS attestation_rewards (
    validator_index BIGINT,
    epoch           BIGINT,
    head_reward     BIGINT,          -- Gwei
    source_reward   BIGINT,          -- Gwei
    target_reward   BIGINT,          -- Gwei
    total_reward    BIGINT,          -- Gwei (head + source + target)
    timestamp       TIMESTAMP,
    PRIMARY KEY ((validator_index), epoch)
) WITH CLUSTERING ORDER BY (epoch DESC)
  AND default_time_to_live = 7776000;
```

**Table: validator_penalties** (slashing and inactivity penalties)

```cql
CREATE TABLE IF NOT EXISTS validator_penalties (
    validator_index BIGINT,
    epoch           BIGINT,
    slot            BIGINT,
    penalty_type    TEXT,            -- 'slashing', 'inactivity_leak', 'attestation_miss'
    penalty_gwei    BIGINT,
    timestamp       TIMESTAMP,
    PRIMARY KEY ((validator_index), epoch, slot)
) WITH CLUSTERING ORDER BY (epoch DESC, slot DESC)
  AND default_time_to_live = 7776000;
```

### 9. ScyllaDB Client Features

- **Shard-aware driver**: Uses `github.com/scylladb/gocql` for optimal shard routing
- **Connection pooling**: Configurable pool size per host
- **Synchronous writes**: Wait for write confirmation with LOCAL_QUORUM consistency
- **Auto-migration**: Creates keyspace and tables on startup if missing
- **Configurable TTL**: Set via config, applied at table level

## Dependencies

| Package | Purpose |

|---------|---------|

| `github.com/rs/zerolog` | High-performance JSON logging |

| `github.com/scylladb/gocql` | ScyllaDB driver with shard-aware routing |

| `golang.org/x/time/rate` | Token bucket rate limiter |

| `gopkg.in/yaml.v3` | YAML config parsing |

## Files to Create

| File | Description |

|------|-------------|

| [`main.go`](main.go) | Entry point with signal handling and graceful shutdown |

| [`go.mod`](go.mod) | Go module definition with dependencies |

| [`config.yaml`](config.yaml) | Example configuration file with ScyllaDB settings |

| [`internal/config/config.go`](internal/config/config.go) | YAML configuration loader |

| [`internal/beacon/client.go`](internal/beacon/client.go) | HTTP client with rate limiting and backoff |

| [`internal/beacon/types.go`](internal/beacon/types.go) | Beacon API response types |

| [`internal/beacon/validators.go`](internal/beacon/validators.go) | Validator status fetching |

| [`internal/beacon/duties.go`](internal/beacon/duties.go) | Attestation duties fetching |

| [`internal/beacon/rewards.go`](internal/beacon/rewards.go) | Attestation rewards fetching |

| [`internal/storage/scylla.go`](internal/storage/scylla.go) | ScyllaDB client and connection management |

| [`internal/storage/migrations.go`](internal/storage/migrations.go) | Schema auto-migration logic |

| [`internal/storage/models.go`](internal/storage/models.go) | Database models for all tables |

| [`internal/storage/repository.go`](internal/storage/repository.go) | Data access layer with CRUD operations |

| [`internal/monitor/monitor.go`](internal/monitor/monitor.go) | Core monitoring orchestration |

| [`internal/monitor/worker.go`](internal/monitor/worker.go) | Worker pool implementation |

| [`internal/monitor/scheduler.go`](internal/monitor/scheduler.go) | Slot-based task scheduling |

| [`pkg/backoff/backoff.go`](pkg/backoff/backoff.go) | Exponential backoff with jitter |