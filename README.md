# adaptive-shaper

A Go agent that manages an HTB queue tree on a MikroTik RB5009 in real time. It
samples traffic over the RouterOS API on a fixed interval classifies each
source and reallocates queue limits to protect latency-sensitive flows when the
uplink comes under load.

## Motivation

I built this because my home uplink fell apart under load. Any sustained upload —
a backup, a game pushing an update, another device seeding — would saturate the
link and my ping would climb from around 15 ms into the hundreds. Throughput
looked fine the whole time; latency did not. That gap is bufferbloat: oversized
buffers along the path hold packets instead of dropping them so a saturated link
keeps moving bytes while every interactive flow waits behind a full queue.

RouterOS already shapes traffic with a queue tree, but the limits are static. I
can hard-cap bulk traffic but a fixed cap is wrong in both directions: set it low
and I waste bandwidth whenever nothing else is running; set it high and it does
nothing at the exact moment contention starts. What I actually wanted was a queue
tree that reacts hand the whole pipe to bulk traffic while the link is idle, and
claw bandwidth back for realtime traffic the instant latency sensitive flows start
competing. That decision has to be made continuously from live measurements rather
than configuredonce which is what this agent does.

The router stays the data plane: it forwards packets and enforces the queue tree.
The agent is the control plane: it reads counters decides and pushes limit
changes back over the API. No traffic passes through the agent so a crash or
restart degrades to whatever limits were last applied rather than dropping the
link.

## How it works

The agent runs a fixed-interval control loop over the RouterOS API:

```
routeros ──► collector ──► classifier ──► controller ──► routeros (SetLimits)
                 │
                 └──► metrics (:9090, Prometheus)
```

- **collector** polls the queue tree and the connection table on every tick and
  computes per-source byte rates from two consecutive samples (a delta divided by
  the elapsed interval).
- **classifier** labels each source. A high UDP-to-total ratio marks it
  `realtime`; a high connection count or a high steady byte cadence marks it
  `bulk`; everything else is `interactive`. The rules are ordered, and
  `interactive` is the catch-all.
- **controller** compares realtime-class utilization against a high watermark and
  shifts HTB limits between classes, with hold-ticks hysteresis so momentary
  spikes do not cause it to oscillate.

Packet marking is done with `/ip firewall mangle` rules; enforcement is a
RouterOS queue tree (HTB). The catch-all `interactive` mangle rule is always last,
because mangle is evaluated top-down and the first match wins.

## Requirements

- Go 1.25
- A MikroTik router with the RouterOS API service enabled
- Docker and Docker Compose (optional — only for the bundled Prometheus/Grafana
  stack)

## Quick start

```bash
git clone https://github.com/dmytroyunyk/Mikrotik-adaptive-shaper.git
cd Mikrotik-adaptive-shaper
cp .env.example .env      # set ROUTEROS_PASSWORD here; .env is gitignored
make docker-run           # agent + Prometheus + Grafana
```

To run the agent alone, without the monitoring stack:

```bash
go run .
```

## Configuration

Configuration lives in `configs/config.yaml`. The router password is the only
secret and is read from the environment (`ROUTEROS_PASSWORD`), never committed.

```yaml
agent:
  interval: 1s              # control-loop period; also the rate-delta window

routeros:
  host: 192.168.88.1
  port: 8728
  username: shaper
  password: ""              # set via ROUTEROS_PASSWORD

shaper:
  interface: ether1
  uplink_mbit: 100          # total budget shared by the queue tree
  realtime_mbit: 50         # starting guarantee for the realtime class
  bulk_mbit: 50             # starting guarantee for the bulk class

classifier:
  udp_ratio_realtime: 0.6   # UDP/total at or above this marks a source realtime
  bulk_min_conns: 20        # this many concurrent connections marks a source bulk
  bulk_min_bps: 2000000     # or this steady byte cadence marks it bulk

controller:
  high_watermark: 0.90      # act when realtime usage reaches this fraction
  hold_ticks: 3             # consecutive ticks required before acting
  step_mbit: 10             # bandwidth moved between classes per action
```

Two thresholds carry most of the tuning and are worth explaining:

- **`high_watermark: 0.90`.** The controller reacts when the realtime class is
  using at least 90% of its current allocation. Reacting at full saturation is too
  late by the time a class is at 100% the queue is already building latency.
  Reacting much earlier (say 0.70) wastes bandwidth by treating normal bursts as
  contention. 0.90 leaves a headroom band wide enough to act before the queue
  fills but narrow enough that ordinary traffic does not trip it.

- **`hold_ticks: 3`.** An action fires only after the utilization stays on one
  side of the watermark for three consecutive ticks. Traffic is bursty at
  subsecond resolution and a controller that reacts to a single tick would
  boost and relax on alternating samples flapping. Every flap rewrites the queue
  tree, and rewriting the tree under load is itself a source of latency. Holding
  for three ticks (three seconds at the default interval) filters transient
  spikes and only responds to sustained pressure. This is asymmetric by design:
  the same counter guards both directions, so a single tick on the opposite side
  resets it and the controller never acts on noise.

## Monitoring

| Service    | URL                     |
|------------|-------------------------|
| Agent      | http://localhost:9090   |
| Prometheus | http://localhost:9091   |
| Grafana    | http://localhost:3000   |

The agent exposes per-class rates current limits and classification counts as
Prometheus metrics on `:9090`. The bundled Grafana dashboard graphs realtime
utilization against the watermark and shows each controller action over time.

## Project layout

```
routeros/    RouterOS API client (mutex-serialized), queue tree, mangle rules
collector/   polls queue + connection stats into a Snapshot each tick
classifier/  assigns a TrafficClass per source from the Snapshot
controller/  decision logic (hysteresis) and limit application via SetLimits
metrics/     Prometheus exporter and HTTP server
config/      YAML loader with environment overrides
models/      shared types (Snapshot, SourceStat, QueueRate, TrafficClass)
main.go      wires the pipeline and handles graceful shutdown
```

## License

Released under the MIT License See [LICENSE](LICENSE).
