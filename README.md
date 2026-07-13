# 🛡️ MikroTik Intelligent Defender

An autonomous network security system written in Go that protects a MikroTik router from attacks in real time. It monitors router logs, detects suspicious activity, automatically blocks attackers at the firewall level, stores attack history, and sends instant Telegram notifications.

---

## Overview

The system is built on a **Data Plane + Control Plane** architecture:

- **Data Plane** (MikroTik RB5009UPr+S+IN) — passes traffic and applies filtering rules (blocks, rate limits).
- **Control Plane** (Go agent on a mini PC) — the brain: analyzes router logs, decides whether to block, and sends commands to the router via the RouterOS API.

```
Internet  →  [ MikroTik Router ]  →  Local Network
                      ↕
               [ Go Agent ]
               "The Brain"
```

When someone attacks the router, the agent detects it, instructs the router to block the attacker, and notifies you on Telegram.

---

## Features

- 📡 **Real-time log monitoring** via the RouterOS API
- 🔍 **Automatic threat detection** — SSH brute-force, port scans, repeated login failures
- ⏱️ **Sliding time-window analysis** — e.g. "10 SSH login failures within 60 seconds → block"
- ✅ **Whitelist protection** — trusted IPs and networks are never blocked
- 💾 **Persistent storage** — attack history saved in SQLite, survives restarts
- 🤖 **Telegram bot** — instant alerts and remote control (`/status`, `/blocked`, `/unban`, `/top`)
- 🔌 **REST API** — programmatic control with API-key authentication
- 📊 **Prometheus metrics** — exported for monitoring and graphing
- 📈 **Grafana dashboard** — visual overview of blocked IPs and attack trends
- 🔄 **Graceful shutdown** — cleanly closes router, database, and bot connections
- 🐳 **Fully containerized** — one command to run the whole stack
- ⚔️ **Attack simulator** — built-in tool to test detection without a real attacker

---

## Architecture

```
Watcher   →  reads MikroTik logs in real time
   ↓
Engine    →  applies detection rules, checks whitelist, decides to block
   ↓
MikroTik  →  blocks the IP via the RouterOS address-list
   ↓
Storage   →  saves the event to SQLite
Bot       →  sends a Telegram notification
Metrics   →  updates Prometheus counters
API       →  exposes data over HTTP
```

---

## Project Structure

```
mikrotik-defender/
├── cmd/
│   ├── agent/        # main entry point — runs the whole system
│   └── bot/          # standalone Telegram bot
│   └── simulator/    # attack simulator for testing detection
├── internal/
│   ├── mikrotik/     # RouterOS API client, watcher, firewall
│   ├── firewall/     # detection engine, rules, whitelist
│   ├── storage/      # SQLite (db, events, queries)
│   ├── bot/          # Telegram bot (handlers, notifier)
│   ├── api/          # REST API (server, handlers, middleware)
│   ├── metrics/      # Prometheus exporter
│   └── config/       # configuration loader with env overrides
├── pkg/utils/        # logger and IP helpers
├── configs/          # config.yml
├── deployments/      # Dockerfile, docker-compose, prometheus, grafana
└── docs/             # architecture and API documentation
```

---

## Tech Stack

`Go 1.25` · `RouterOS API` · `SQLite` · `Gin` · `telebot.v3` · `Prometheus` · `Grafana` · `Docker` · `Swagger/OpenAPI`

---

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/dmytroyunyk/Mikrotik-firewall.git
cd Mikrotik-firewall
```

### 2. Configure secrets

Create a `.env` file in the project root:

```env
MIKROTIK_PASSWORD=your_router_password
TELEGRAM_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id
API_KEY=your_api_key
```

> - Get the Telegram token from [@BotFather](https://t.me/BotFather)
> - Get your chat ID from [@userinfobot](https://t.me/userinfobot)

### 3. Adjust the config

Edit `configs/config.yml` with your router address and firewall thresholds.

### 4. Run with Docker

```bash
make docker-run
```

This starts the agent, Prometheus, and Grafana.

### 5. Run locally (without Docker)

```bash
env $(cat .env) go run ./cmd/agent
```

---

## Configuration

```yaml
mikrotik:
  address: "192.168.88.1:8728"   # router IP + API port
  username: "admin"
  password: ""                   # set via MIKROTIK_PASSWORD

firewall:
  ban_threshold: 10              # events before blocking
  ban_duration_minutes: 60       # how long to block
  whitelist:
    - "192.168.88.0/24"          # never blocked
    - "127.0.0.1"

telegram:
  token: ""                      # set via TELEGRAM_TOKEN
  chat_id: ""                    # set via TELEGRAM_CHAT_ID

storage:
  path: "./data/defender.db"

api:
  port: 8080
  key: ""                        # set via API_KEY

log:
  level: "info"                  # debug / info / warn / error
```

> Secrets are read from environment variables and never committed to the repository.

---

## Telegram Commands

| Command        | Description                     |
|----------------|---------------------------------|
| `/start`       | Welcome message and command list |
| `/status`      | System statistics               |
| `/blocked`     | List of currently blocked IPs   |
| `/top`         | Top 10 attackers                |
| `/unban <IP>`  | Unblock an IP address           |

---

## REST API

All `/api/v1/*` endpoints require the `X-API-Key` header.

| Method   | Endpoint               | Description              |
|----------|------------------------|--------------------------|
| `GET`    | `/health`              | Liveness check (no auth) |
| `GET`    | `/metrics`             | Prometheus metrics       |
| `GET`    | `/api/v1/stats`        | System statistics        |
| `GET`    | `/api/v1/blocked`      | List of blocked IPs      |
| `DELETE` | `/api/v1/blocked/{ip}` | Unblock an IP            |
| `GET`    | `/api/v1/events`       | Recent attack events     |
| `GET`    | `/api/v1/attackers`    | Top attackers            |

## Attack Simulator
 
A built-in tool to safely test the detection system against your own router. It generates SSH brute-force attempts or port scans so you can verify that the engine detects and blocks them — no real attacker required.
 
```bash
# Simulate an SSH brute-force attack (20 attempts)
go run ./cmd/simulator --mode ssh --target 192.168.88.1:22 --count 20 --delay 200
 
# Simulate a port scan
go run ./cmd/simulator --mode scan --target 192.168.88.1 --count 20
```
 
| Flag       | Description                              | Default              |
|------------|------------------------------------------|----------------------|
| `--target` | Router IP:port to test                   | `192.168.88.1:22`    |
| `--mode`   | Attack type: `ssh` or `scan`             | `ssh`                |
| `--count`  | Number of attempts                       | `15`                 |
| `--delay`  | Delay between attempts (milliseconds)    | `500`                |
 
> ⚠️ Run the simulator from a device **outside** the whitelist, otherwise the source IP will never be blocked. Use it only against your own equipment.
 
---

**Example:**

```bash
curl -H "X-API-Key: your_key" http://localhost:8080/api/v1/stats
```

Interactive API documentation (Swagger UI) is available at:
```
http://localhost:8080/swagger/index.html
```

---

## Monitoring

| Service    | URL                                        |
|------------|--------------------------------------------|
| Prometheus | http://localhost:9090                      |
| Grafana    | http://localhost:3000 (admin / admin)      |

The Grafana dashboard loads automatically and shows blocked IPs, total events, events in the last 24 hours, and blocking trends over time.

---

## Makefile Commands

```bash
make build          # build the agent binary
make run            # run the agent
make test           # run all tests
make docker-run     # start the full stack with Docker
make docker-stop    # stop all containers
make docker-logs    # follow container logs
```

---

## Testing

```bash
go test ./... -v
```

Unit tests cover the firewall engine, detection rules, whitelist logic, configuration loading, database operations, and utility functions using table-driven tests.

---

## License

[MIT](LICENSE)