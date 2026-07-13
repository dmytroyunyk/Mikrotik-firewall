# Architecture

This document describes the internal design of MikroTik Intelligent Defender: how components interact, how data flows through the system, and how concurrency is handled.

---

## Design Philosophy

The system separates responsibilities into two planes, a pattern borrowed from network engineering:

- **Data Plane** (MikroTik router) — forwards traffic and enforces filtering rules. It does not make decisions; it only executes them.
- **Control Plane** (Go agent) — observes, analyzes, and decides. It never touches traffic directly. Instead, it instructs the Data Plane through the RouterOS API.

This separation means the agent can run on cheap, separate hardware (a mini PC) while the router focuses purely on packet forwarding. It also means the agent can be restarted, updated, or scaled without interrupting network traffic.

---

## Component Overview

```
┌─────────────────────────────────────────────────────────┐
│                   Control Plane (Go)                     │
│                                                          │
│  ┌─────────┐  events   ┌────────┐  block   ┌──────────┐ │
│  │ Watcher │─────────► │ Engine │─────────► │ MikroTik │ │
│  └─────────┘ (channel) └────────┘  (API)   │  Client  │ │
│                             │               └──────────┘ │
│                             │ blocked IP                  │
│                             ▼                             │
│                        ┌─────────┐                        │
│                        │ Storage │                        │
│                        │(SQLite) │                        │
│                        └─────────┘                        │
│                             │                             │
│                   ┌─────────┴──────────┐                  │
│                   ▼                    ▼                  │
│             ┌──────────┐        ┌──────────┐             │
│             │ Telegram │        │ Metrics  │             │
│             │   Bot    │        │(Prometheus)            │
│             └──────────┘        └──────────┘             │
│                                      │                    │
│                               ┌──────────┐               │
│                               │ REST API │               │
│                               └──────────┘               │
└──────────────────────────────────────────────────────────┘
         │ read logs (API)            │ block (API)
         ▼                            ▼
┌──────────────────────────────────────────────────────────┐
│                  Data Plane (MikroTik)                    │
└──────────────────────────────────────────────────────────┘
```

---

## Data Flow

The lifecycle of a single attack, step by step:

1. **Attacker probes the router** — for example, repeated SSH login attempts.
2. **Router logs the event** — MikroTik writes it to its internal log stream.
3. **Watcher reads the log** — `mikrotik.Watcher` subscribes to the router's log stream over the RouterOS API, parses each line, extracts the source IP and event type, and pushes a `LogEntry` into a Go channel.
4. **Engine receives the event** — `firewall.Engine` reads from the channel. It first checks the whitelist; whitelisted IPs are ignored immediately. It then finds a matching detection rule for the event type.
5. **Engine counts events in a sliding window** — for each IP + event type pair, the engine keeps a list of timestamps. Old timestamps outside the rule's time window are dropped. If the number of recent events reaches the rule's threshold, the IP is blocked.
6. **MikroTik Client blocks the IP** — `mikrotik.Client.BlockIP` sends an API command that adds the IP to the router's blacklist address-list with a timeout.
7. **Event is persisted** — `storage` writes both the event and the blocked-IP record to SQLite.
8. **Notifications fire** — the Telegram bot sends an alert, and the Prometheus counter is incremented.
9. **Data becomes queryable** — the REST API and Telegram commands can now report the block.

---

## Package Responsibilities

### `internal/mikrotik` — bridge to the router

| File | Responsibility |
|------|---------------|
| `client.go` | Manages the RouterOS API connection (connect, disconnect, health check) |
| `firewall.go` | Block, unblock, and list blocked IPs via the address-list |
| `watcher.go` | Subscribes to the router's log stream and parses entries into structured events |

### `internal/firewall` — decision-making core

| File | Responsibility |
|------|---------------|
| `whitelist.go` | Checks whether an IP falls within a trusted single IP or CIDR range |
| `rules.go` | Defines detection rules (threshold + time window per event type) |
| `engine.go` | Combines rules, whitelist, and the sliding-window counter. The "block or not" decision happens here. Safe for concurrent use via a mutex |

### `internal/storage` — persistence and analytics

| File | Responsibility |
|------|---------------|
| `db.go` | Opens the SQLite database and creates tables |
| `events.go` | Saves and reads events and blocked-IP records |
| `queries.go` | Statistics: top attackers, attack counts, block status |

### `internal/bot` — human interface

| File | Responsibility |
|------|---------------|
| `bot.go` | Initializes the Telegram bot and registers commands |
| `handlers.go` | Responds to user commands |
| `notifier.go` | Sends automatic alerts on block, error, startup, and shutdown |

### `internal/api` — programmatic interface

| File | Responsibility |
|------|---------------|
| `server.go` | HTTP server and route registration |
| `handlers.go` | Request handlers for each endpoint |
| `middleware.go` | API-key authentication and request logging |

### `internal/metrics`

| File | Responsibility |
|------|---------------|
| `prometheus.go` | Defines and exports gauges and counters for Prometheus to scrape |

### `internal/config`

| File | Responsibility |
|------|---------------|
| `config.go` | Loads `config.yml`, validates required fields, and overrides secrets from environment variables |

### `pkg/utils`

| File | Responsibility |
|------|---------------|
| `logger.go` | Structured logging built on `slog` |
| `ip.go` | IP validation, private-range detection, sanitization, and log parsing |

### `cmd/simulator` — attack simulator
 
| File | Responsibility |
|------|---------------|
| `main.go` | Generates SSH brute-force attempts and port scans against a target router to test the detection pipeline end to end |

---

## Concurrency Model

The system relies on Go's goroutines and channels rather than shared locks wherever possible.

```
main goroutine
   │
   ├── go watcher.Watch()       — reads logs from the router
   ├── go processEvents()       — handles events one at a time
   ├── go apiServer.Start()     — serves HTTP
   ├── go bot.Start()           — Telegram polling
   └── go metricsUpdater()      — refreshes Prometheus metrics
   │
   └── <-quit  (blocks until Ctrl+C / SIGTERM)
          │
          └── close(stop) → all goroutines return cleanly
              defer: close router, db, bot connections
```

- **Watcher** runs in its own goroutine, continuously reading the router's log stream without blocking the main flow. Results are pushed into a buffered `chan LogEntry`.
- **Event-processing loop** reads from the channel with `for event := range events`, processing one event at a time.
- **API server, Telegram bot, and metrics updater** each run in their own goroutines and do not block one another.
- **Shared state is protected by a mutex.** The engine's sliding-window counters are guarded by `sync.Mutex`, preventing concurrent map corruption.
- **Channels signal shutdown.** On `SIGINT` or `SIGTERM`, the main goroutine closes a `stop` channel. Each background goroutine watches it with a `select` and returns cleanly.

---

## Sliding Time Window

The core detection mechanism. Instead of a simple counter, the engine tracks *when* each event happened.

For a rule like **"10 SSH failures in 60 seconds"**:

1. Each new event appends the current timestamp to a list keyed by `IP + event type`.
2. On every new event, timestamps older than 60 seconds are removed.
3. If the remaining list has **10 or more entries**, the rule fires.

This gives an accurate, rolling measure of recent activity. An attacker who spreads attempts over hours never triggers the rule, while a burst of 10 attempts in a few seconds does. Old entries are cleaned automatically, so memory does not grow without bound.

---

## Attack Simulator
 
The `cmd/simulator` tool exercises the full detection pipeline without needing a real attacker. It is a standalone binary that talks to the router directly, not part of the running agent.
 
- **SSH mode** repeatedly opens SSH connections with deliberately wrong passwords. Each failed attempt makes the router write a `login failure` entry to its log stream, which the Watcher then picks up.
- **Scan mode** opens TCP connections to a list of common ports, producing connection events the firewall can flag as scanning activity.
Because whitelisted addresses are never blocked, the simulator must be run from a source IP that is **outside** the configured whitelist. This makes it a safe, repeatable way to confirm that detection rules, the sliding window, blocking, storage, and notifications all work together correctly.
 
---

## Security Considerations

- **Secrets never live in code.** Passwords, tokens, and API keys are read from environment variables at runtime and excluded from version control via `.gitignore`.
- **The REST API requires authentication.** Every `/api/v1/*` route is guarded by middleware that validates an `X-API-Key` header. Missing or invalid keys receive `401 Unauthorized`.
- **The whitelist prevents self-lockout.** Local networks and localhost are checked before any block decision, so the system can never lock out the administrator or trusted devices.
- **The router config is mounted read-only in Docker**, preventing accidental modification from within the container.

---

## Deployment

The entire stack — agent, Prometheus, and Grafana — is defined in a single `docker-compose.yml` and started with one command.

The agent image uses a **multi-stage build**:
1. The first stage compiles the binary with the full Go toolchain.
2. The second stage copies only the binary into a minimal Alpine image, keeping the final image small.

Grafana and Prometheus are **provisioned automatically** — the datasource and dashboard are configured on startup with no manual steps required.