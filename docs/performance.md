# SYN scan vs connect scan: measured

Every number in this document came from a run executed on the machine described
below. Nothing here is projected, extrapolated, or taken from another tool's
published results. Where something could not be measured, it says so instead of
guessing.

**Summary:** the SYN scanner is **1.75x** faster than the connect scanner on
65,535 loopback ports, and **3.0x** at `--concurrency 2000`. It is level with it
against a filtered host, where both are bound by the same timeout arithmetic.
It also does not consume a file descriptor per port, so it still finds every
open port at concurrency levels where the connect scan silently misses up to a
quarter of them.

The first working version was *slower* than the connect scan (0.75x). What fixed
it was not the packet library — that is 0.5% of the runtime — but sending from
several raw sockets instead of one, because the kernel serializes writes per
socket. The measurements behind that are in
[Why the first version was slow](#why-the-first-version-was-slow).

---

## The problem

`netdiag scan` dials every port with `net.DialTimeout`, completing a full TCP
three-way handshake and then closing the connection. That costs a socket and a
file descriptor per port, and leaves a completed connection in the target's
logs.

A SYN scan sends only the initial SYN and reads the reply: SYN-ACK means open,
RST means closed, silence means filtered. The handshake is never completed.

The expectation going in was a speed win of an order of magnitude. What was
measured is 1.75x on loopback, rising to 3.0x at high concurrency — real, but
nothing like the numbers SYN scanners are usually quoted at, for reasons the
caveats section spells out.

## Environment

| | |
|---|---|
| CPU | 12th Gen Intel Core i7-12650H, 16 logical CPUs |
| Kernel | Linux 7.0.0-28-generic x86_64 |
| `net.core.rmem_max` | 4194304 |
| `ulimit -n` | 1048576 (lowered deliberately in scenario 4) |
| Privilege | `setcap cap_net_raw+ep` on the binary, not root |
| netdiag | this branch, `go build` |

These numbers are measured **on the host, on bare metal**. An earlier revision
of this document reported container measurements, because the host had no
`CAP_NET_RAW`; granting the binary the capability directly gives a cleaner
measurement, and a faster one — the same scans inside a `--cap-add=NET_RAW`
container measured 1.45x and 1.9x, against 1.75x and 3.0x here. The container's
network namespace, not the scanner, accounts for the difference.

## Method

```bash
go build -o /tmp/netdiag .
sudo setcap cap_net_raw+ep /tmp/netdiag
/tmp/netdiag scan 127.0.0.1 -p 1-65535 --benchmark
```

`--benchmark` runs both scanners against the same target in the same process,
one after the other, and prints the comparison. Five runs per scenario unless
noted; the tables report the median. Both scanners see identical port lists,
timeouts and concurrency.

Targets are entirely local: `127.0.0.1`, and `172.17.0.99`, an address on this
machine's own `docker0` bridge that nothing answers for. No public internet
traffic is involved in any measurement here.

---

## Scenario 1 — 65,535 closed ports on loopback, `--concurrency 100`

| Method | Duration (median) | Ports/sec (median) | Speedup |
|---|---|---|---|
| connect | 264 ms | 248,496 | 1.0x (baseline) |
| syn | 151 ms | 433,686 | **1.75x** |

Both methods found the same 4 open ports, which is the cross-check that matters:
a faster scanner that disagreed with the slower one would not be a faster
scanner.

## Scenario 2 — 65,535 closed ports on loopback, `--concurrency 2000`

| Method | Duration (median) | Ports/sec (median) | Speedup |
|---|---|---|---|
| connect | 385 ms | 170,308 | 1.0x (baseline) |
| syn | 128 ms | 510,140 | **3.0x** |

Higher concurrency makes the connect scan *slower* — more goroutines contending
over more sockets on the same loopback path — while the SYN scanner gets faster,
because its cost is syscalls it can spread across a fixed set of sockets rather
than sockets it must open one per port. The gap widens with concurrency, which
is the shape you would expect from the descriptor argument in scenario 4.

## Scenario 3 — 1,024 filtered ports, `--timeout 1s`

Target `172.17.0.99`, an unassigned address on this machine's `docker0` bridge.
Nothing answers, so every port is filtered and every probe waits out its
timeout.

| Method | `-c 100` (median of 3 runs) |
|---|---|
| connect | 11.009 s |
| syn | 11.008 s |
| speedup | **1.0x** |

Both methods are bound by the same arithmetic: ports ÷ concurrency × timeout.
Nothing about half-open probing changes that when the target is silent.

## Scenario 4 — 200 open ports under file descriptor pressure

200 listeners on `127.0.0.1:20000-20199`, opened by a separate helper process.
The scan is run with a reduced `ulimit -n`, which is what a scan of a large
range looks like on a machine with a normal descriptor limit.

**`ulimit -n 32`, `--concurrency 500`, 5 runs — open ports found (200 exist):**

| Run | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|
| connect | 200 | 184 | 166 | 154 | 200 |
| syn | 200 | 200 | 200 | 200 | 200 |

The connect scan missed open ports in three of five runs, reporting as few as
154 of 200 — a 23% false negative rate, presented as a clean successful scan. A
`dial` that fails with `EMFILE` is indistinguishable, to that code, from a
closed port. The SYN scanner's socket count is fixed and independent of the port
count, so its results do not degrade with the descriptor limit.

The size of the shortfall depends on how many dials are genuinely simultaneous,
so treat the exact figure as variable and the direction as the finding. At a
more generous `ulimit -n 64` with `--concurrency 200` the effect is
intermittent: one run in six reported 190 of 200, the rest found all 200.

The durations in this scenario are 2–4 ms and too noisy to compare; no speed
claim is made from it.

---

## Why the first version was slow

The first working SYN scanner took 359 ms where the connect scan took 269 ms.
Rather than guess, each part of the send path was timed on its own, against
`127.0.0.1`, 65,535 packets per measurement.

| Component | Time for 65,535 packets | Share of the 359 ms scan |
|---|---|---|
| Build header + compute checksum, no syscalls | 1.7 ms | 0.5% |
| Write them to **one** raw socket, one goroutine | 192 ms | 53% |
| Everything else (pacing, receiving, correlating) | ~165 ms | 46% |

**The packet library is not the bottleneck.** Building and checksumming all
65,535 packets costs 1.7 ms — half a percent of the scan. Swapping packet
libraries, or hand-rolling the header, cannot move a number that small. The
scan is dominated by syscalls, not by CPU work in userspace.

**One write syscall per packet, on one thread, was the bottleneck.** 192 ms of
`sendto` calls is already 71% of the connect scan's entire runtime, before the
SYN scanner does anything else. The connect scanner issues its syscalls from a
hundred goroutines that the Go runtime spreads over 16 CPUs; the first SYN
scanner issued them from one.

The obvious fix does not work. Adding sender goroutines to the *same* socket
makes it worse, because the kernel serializes writes per socket and the extra
goroutines only add contention:

| Sender goroutines, one shared socket | 1 | 2 | 4 | 8 | 16 |
|---|---|---|---|---|---|
| Time for 65,535 packets | 156 ms | 166 ms | 178 ms | 218 ms | 221 ms |

Giving each sender **its own** socket is what parallelizes, because the
serialization is per socket rather than per device:

| Independent raw sockets | 1 | 4 | 8 |
|---|---|---|---|
| Time for 65,535 packets | 185 ms | 53 ms | 39 ms |

The scanner now sends from eight raw sockets and receives on one. That took the
65,535-port scan from 359 ms to 199 ms in the container where the problem was
diagnosed, and to 151 ms on the host — turning 0.75x into 1.75x. Pacing and the
in-flight queue stay on a single goroutine that owns them without locks; the
workers do nothing but build a packet and make the write call.

**The receiver still sees traffic it did not ask for.** A raw `ip4:tcp` socket
is unfiltered: it receives every TCP segment on the machine, including the SYNs
this scanner just sent. In one instrumented run the receiver read 21,061 of its
own outbound SYNs alongside the replies it wanted. Attaching a BPF filter to the
socket would remove that work, and is the obvious next thing to try; it has not
been done, so no claim is made about what it would save.

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
- The component timings in "Why the first version was slow" were taken inside a
  `--cap-add=NET_RAW` container, where that work was done. The scenario tables
  are all from the host. The two are not directly comparable, which is why the
  container's own scan figures are quoted alongside the host's rather than mixed
  into them.
- Scenario 4 depends on the descriptor limit and on how many dials are genuinely
  simultaneous, which is why the effect is severe at `ulimit -n 32` and
  intermittent at 64.

## Conclusion

`--fast` is 1.75x the connect scan on 65,535 loopback ports and 3.0x at
`--concurrency 2000`, level with it against a filtered host, and more accurate
than it under file descriptor pressure. It never completes a handshake against
the target.

That is well short of the order-of-magnitude figures SYN scanners are usually
quoted at, and the reason is the target rather than the implementation: on
loopback there is no timeout to avoid and no round trip to overlap, which is
exactly what a SYN scan exists to exploit. The measurement that would show that
advantage needs a remote host, and this environment does not have one to
scan.
