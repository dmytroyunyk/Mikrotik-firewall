# Architecture

This document explains the design decisions behind adaptive shaper why it is
structured as a polling control loop how concurrency is handled how it survives
RouterOS API failures, and how the HTB reallocation and hysteresis actually work.

## Control plane and data plane

The router is the data plane. It forwards packets and enforces the queue tree it
is the only component in the packet path. The agent is the control plane. It
observes counters and pushes limit changes, but no user traffic ever passes
through it. This split is deliberate: the failure of a control plane should
degrade behaviour, not connectivity. If the agent crashes, the queue tree keeps
running with whatever limits were last written, and the link is unaffected.

## Data flow

```
routeros ──► collector ──► classifier ──► controller ──► routeros (SetLimits)
                 │
                 └──► metrics
```

Each tick, the collector produces one immutable `Snapshot` describing the state
of the link at that instant per source stats and per queue rates. The Snapshot
flows through classification, then fans out to the metrics exporter and the
controller. Nothing downstream mutates it so there is no shared state to guard
along the pipeline — the only shared resource in the whole system is the single
RouterOS connection, which is discussed below.

## Why a control loop, not event-driven

Shaping is a problem about aggregate rate over time, not about discrete events. The
question the agent answers is the realtime class under pressure right now? is
only meaningful when averaged over a window. There is no single packet whose
arrival should trigger a reallocation.

A fixed-interval control loop fits this directly, and the RouterOS API reinforces
the choice:

- **The API is pull not push.** RouterOS exposes counters you read it does not
  stream per-packet or per-connection events. An event-driven design would have to
  synthesise events by polling anyway, so polling is the honest primitive.
- **Rate is a delta between samples.** Per source byte rates are computed as
  `(bytes_now - bytes_prev) / interval` That calculation only exists because
  there are two samples a fixed time apart. The loop period is both the control
  period and the measurement window one interval serves both.
- **Bounded work per tick.** A loop does a predictable amount of work each period
  regardless of traffic volume. An event driven shaper reacting to connection
  churn would do unbounded work under exactly the load it most needs to survive 
  a DDoS or a torrent swarm opening thousands of connections would become a
  self inflicted overload.
- **Hysteresis needs a clock.** "Sustained for three ticks" only has meaning in a
  system with ticks. The loop gives hold-ticks a natural unit.

The trade off is latency of reaction the controller can be up to one interval
behind reality. At a one-second interval that is acceptable, because the thing it
protects against — bufferbloat building in a saturated queue develops over
seconds not milliseconds. If sub second reaction were ever required the answer
is a shorter interval not an event model.

## Concurrency: mutex and channels, each where it fits

The system uses both `sync.Mutex` and channels and the split follows the standard
Go guidance share memory by communicating for data flow guard shared memory
with a mutex for a shared resource.

**Channels move Snapshots between stages.** The collector owns a Snapshot, then
hands it off; once it is on the channel, the collector is done with it and the next
stage owns it. This is ownership transfer, which is what channels are for. The
pipeline (collector classifier fan out to metrics and controller) is wired
with unbuffered channels, and every send is paired with a `ctx.Done()` case so a
blocked send cannot wedge shutdown:

```go
select {
case toController <- snap:
case <-ctx.Done():
    return
}
```

**A mutex guards the one RouterOS connection.** The agent holds a single TCP
connection to the router's API. That connection is a request/response protocol a
call writes a request and then reads its response and a second goroutine writing
its own request in between would interleave two dialogues on one socket and
corrupt both. The mutex therefore wraps the *entire* request-response cycle not
just field access:

```go
func (c *Client) run(ctx context.Context, args ...string) (*routeros.Reply, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    // full write-request / read-reply dialogue happens under the lock
}
```

This is the case a mutex is designed for: a single shared resource that must be
used by one goroutine at a time. Trying to model it with a channel for instance
a request/response channel pair fronting the connection would reinvent a lock
with more moving parts and no benefit. The rule the codebase follows is: channels
for handing data between stages, a mutex for the resource those stages share.

## RouterOS client and connection handling

The client connects once at startup. A failure to connect there is fatal there
is nothing useful the agent can do without a link to the router so it exits with
a clear error rather than starting a loop that can never succeed.

During the loop every API call returns its error up the stack. The mutex is what
makes this safe: because each call holds the lock for its whole dialogue a call
that fails partway through releases the lock in a clean state (via `defer`), and
the next call starts a fresh dialogue rather than reading a stale half-response.
An error does not leave the connection in an ambiguous state for other goroutines.

Errors propagate to the stage's `Run` loop, which surfaces them through the agent.
Shutdown is driven by context: `signal.NotifyContext` cancels on SIGTERM/SIGINT,
every loop selects on `ctx.Done()`, and the connection is closed with a deferred
`Close()`. This gives an orderly teardown — no goroutine is left blocked on a
channel send or a socket read.

**Boundary automatic reconnection is not implemented.** Today a dropped
connection surfaces as a call error and stops the affected loop; recovery is a
process restart (the container restart policy handles this in the bundled
deployment). This is a deliberate boundary for the current version, not an
oversight fail-fast with an external supervisor is simpler and more predictable
than an in process reconnect state machine, and it avoids the harder question of
what limits to reassert after a gap. A reconnect ith backoff path that
re establishes the connection and re applies the last known limits is the natural
next step and is tracked as future work.

## Classification

Classification turns raw per ource stats into a `TrafficClass`. The rules are
ordered and the first match wins so ordering is part of the logic not an
implementation detail:

1. **realtime** — the source's UDP to total ratio is at or above
   `udp_ratio_realtime`. Interactive real time traffic (calls games) is
   predominantly UDP and is the traffic that suffers most from bufferbloat so it
   is matched first.
2. **bulk** — the source has either a high concurrent connection count
   (`bulk_min_conns`) or a high steady byte cadence (`bulk_min_bps`). Either
   signal on its own is enough: swarms show up as connection count single large
   transfers show up as cadence.
3. **interactive** — everything else. This is the catch all and is always last.

The ratio test is guarded against a source with zero connections so the division
cannot produce a NaN. The same ordering is mirrored in the mangle rules on the
router, where the `interactive` catch-all is likewise the final rule.

## HTB and the control decision

Enforcement is a RouterOS queue tree, which implements HTB (Hierarchical Token
Bucket). In HTB each queue has two rates: a guaranteed rate (`limit-at`, the
bandwidth it is always entitled to) and a ceiling (`max-limit`, the most it may
use when siblings leave headroom). A parent queue holds the total uplink budget;
the `realtime`, `bulk`, and `interactive` children draw from it. Packets are
directed into a child by the packet mark that the mangle rules set. Reallocating
bandwidth between classes therefore means changing these `limit-at`/`max-limit`
values — which is exactly what the controller does through `SetLimits`.

The decision itself is a small state machine driven by realtime-class
utilization:

```
if utilization >= high_watermark:
        tightTicks++;  freeTicks = 0
        if tightTicks >= hold_ticks: reset; BoostRT
else:
        freeTicks++;   tightTicks = 0
        if freeTicks >= hold_ticks: reset; RelaxRT
```

- **BoostRT** raises the realtime guarantee by `step_mbit` and lowers bulk by the
  same amount — bandwidth is moved toward realtime when it is under sustained
  pressure.
- **RelaxRT** does the reverse returning bandwidth to bulk once realtime has been
  comfortably below the watermark for long enough.

Every shift is bounded. A class is never pushed below a floor (`minMbit`, 50) and
the sum of guarantees is held within the uplink budget, so the controller cannot
starve a class or over-commit the link no matter how long pressure persists.

### Hysteresis and why it matters

The two counters, `tightTicks` and `freeTicks`, are the hysteresis. An action
fires only after utilization stays on one side of the watermark for `hold_ticks`
consecutive samples, and the opposing counter is reset every tick. A single tick
on the other side of the line zeroes the count and the pending action is
abandoned.

Without this, a source hovering right at 90% would push utilization across the
watermark on alternating samples and the controller would issue BoostRT, RelaxRT,
BoostRT on successive ticks flapping. Flapping is not just wasted work: each
action rewrites the queue tree, and rewriting the tree while it is actively
shaping a saturated link perturbs the very queues it is trying to stabilise
adding the latency the whole system exists to remove. Hysteresis converts a noisy,
sub second utilization signal into a small number of deliberate actions that track
real sustained shifts in load.
