# SYN scan vs connect scan: measured

Every number in this document came from a run executed on the machine described
below. Nothing here is projected, extrapolated, or taken from another tool's
published results. Where something could not be measured, it says so instead of
guessing.

**Summary:** on the hardware and targets available here, the SYN scanner is
**not faster** than the connect scanner — it ranges from 0.75x to 1.05x. Its
measured advantage is correctness, not speed: it does not consume a file
descriptor per port, so it still finds every open port at concurrency levels
where the connect scan silently misses up to a third of them.

---

## The problem

`netdiag scan` dials every port with `net.DialTimeout`, completing a full TCP
three-way handshake and then closing the connection. That costs a socket and a
file descriptor per port, and leaves a completed connection in the target's
logs.

A SYN scan sends only the initial SYN and reads the reply: SYN-ACK means open,
RST means closed, silence means filtered. The handshake is never completed. The
expectation going in was a large speed win. The measurements did not support
that, and this document reports what actually happened.

## Environment

| | |
|---|---|
| CPU | 12th Gen Intel Core i7-12650H, 16 logical CPUs |
| Host kernel | Linux 7.0.0-28-generic x86_64 |
| Host OS | Linux, `CapPrm: 0`, `net.ipv4.ping_group_range = 1 0` |
| Container | `golang:1.24`, `--cap-add=NET_RAW` |
| `net.core.rmem_max` | 4194304 |
| netdiag | this branch, `go build -buildvcs=false` |

The scans run **inside a Docker container**, because the host has no
`CAP_NET_RAW` and `sudo` requires a password that is not available to the
automated environment. A raw socket cannot be opened on the host at all, so the
SYN path there always falls back to the connect scan.

## Method

```bash
docker run --rm --cap-add=NET_RAW -v "$PWD:/src" -w /src golang:1.24 sh -c \
  'go build -buildvcs=false -o /tmp/netdiag . && /tmp/netdiag scan 127.0.0.1 -p 1-65535 --benchmark'
```

`--benchmark` runs both scanners against the same target in the same process,
one after the other, and prints the comparison. Five runs per scenario unless
noted; the tables report the median. Both scanners see identical port lists,
timeouts and concurrency.

Targets are entirely local: `127.0.0.1`, and `172.17.0.99`, an address on the
container's own bridge network that no host answers for. No public internet
traffic is involved in any measurement here.

---

## Scenario 1 — 65,535 closed ports on loopback, `--concurrency 100`

| Method | Duration (median) | Ports/sec (median) | Speedup |
|---|---|---|---|
| connect | 269 ms | 243,565 | 1.0x (baseline) |
| syn | 359 ms | 182,514 | **0.75x** |

The SYN scan is *slower*. On loopback a `connect()` to a closed port returns
`ECONNREFUSED` immediately — there is no timeout to avoid and no handshake to
save, because the handshake never gets past the first packet. Meanwhile the SYN
scanner pays for a userspace sender loop, a per-packet checksum, and a receiver
that must inspect every TCP segment on the machine, including its own outbound
SYNs and the kernel's RST replies.

## Scenario 2 — 65,535 closed ports on loopback, `--concurrency 2000`

| Method | Duration (median) | Ports/sec (median) | Speedup |
|---|---|---|---|
| connect | 358 ms | 183,139 | 1.0x (baseline) |
| syn | 375 ms | 174,874 | **0.95x** |

Raising concurrency does not help either method here; both are already limited
by how fast loopback can turn packets around.

## Scenario 3 — 1,024 filtered ports, `--timeout 1s`

Target `172.17.0.99`, an unassigned address on the container bridge. Nothing
answers, so every port is filtered and every probe waits out its timeout.

| Method | `-c 100` (3 runs) | `-c 2000` (2 runs) |
|---|---|---|
| connect | 11.009 s | 1.058 s |
| syn | 11.008 s | 1.005 s |
| speedup | **1.0x** | **1.05x** |

Both methods are bound by the same arithmetic: ports ÷ concurrency × timeout.
Nothing about half-open probing changes that when the target is silent.

## Scenario 4 — 200 open ports under file descriptor pressure

200 listeners on `127.0.0.1:20000-20199`, opened by a separate helper process.
The scan is run with a reduced `ulimit -n`, which is what a scan of a large
range looks like on a machine with a normal descriptor limit.

**`ulimit -n 32`, `--concurrency 500`, 5 runs — open ports found (200 exist):**

| Run | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|
| connect | 128 | 196 | 186 | 200 | 169 |
| syn | 200 | 200 | 200 | 200 | 200 |

The connect scan missed open ports in four of five runs, once reporting only
128 of 200 — a 36% false negative rate, reported as a clean successful scan. A
`dial` that fails with `EMFILE` is indistinguishable, to that code, from a
closed port. The SYN scanner uses one raw socket for the entire scan, so its
results do not degrade with the descriptor limit.

At a more generous `ulimit -n 64` with `--concurrency 200` the effect is
intermittent: one run in six reported 190 of 200, the other five found all 200.
The durations in this scenario are 2–4 ms and too noisy to compare; no speed
claim is made from it.

---

## What the measurements changed in the code

Two defects were found by benchmarking, not by testing:

1. **AIMD collapsed on filtered ranges.** The first version halved the
   in-flight limit for any window containing a timeout. A filtered range is
   100% timeouts, so the limit fell to its minimum and stayed there, making a
   filtered scan roughly eight times *slower* than the connect scan. Fixed by
   backing off only on windows that contain both replies and timeouts: total
   silence is a filtered host, not congestion, and slowing down recovers
   nothing.
2. **The backoff floor was set too high.** With a floor of `concurrency/8`, a
   `-c 2000` scan of 65,535 loopback ports took **2m08s** — 350x worse than the
   same scan at `-c 100`. Above a certain rate the raw socket's receive buffer
   overflows, replies are dropped, and every probe waits its full timeout
   instead of a round trip; the floor kept the scanner pinned in exactly that
   regime. Fixed with a small absolute floor plus a 4 MiB receive buffer, after
   which the same scan runs in 360 ms.

## Caveats

- **Loopback is not a network.** Every timing here is from a target with
  effectively zero round-trip time, no packet loss, no middleboxes and no rate
  limiting. The case where a SYN scan is conventionally expected to win — a WAN
  target that silently drops packets to closed ports, so the connect scan pays a
  full timeout per port while the SYN scanner keeps thousands of probes in
  flight — **was not measured here**, because this environment has no
  authorized remote target to scan. Treat it as untested rather than as a
  claimed result.
- The connect scan's numbers depend heavily on `--timeout` whenever ports are
  filtered. Scenario 3 is as much a measurement of the timeout as of either
  method.
- Container measurements include the bridge network and container namespace;
  they are not identical to bare metal.
- Scenario 4 depends on the descriptor limit and on how many dials are genuinely
  simultaneous, which is why the effect is severe at `ulimit -n 32` and
  intermittent at 64.

## Conclusion

The SYN scanner did not deliver the speed win this phase set out to get, on the
targets that could be measured. What it does deliver, measurably, is a scan
whose accuracy does not depend on the process file descriptor limit, and one
that never completes a handshake against the target. On this hardware, against
loopback, `--fast` is a correctness feature with a small speed cost.
