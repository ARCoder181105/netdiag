# netdiag demo recording.
#
# Produces docs/demo.gif, which the README references.
#
# Linux only, because of the setcap step:
#
#   go build -o /tmp/netdiag . && sudo setcap cap_net_raw+ep /tmp/netdiag
#   vhs docs/demo.tape
#
# On macOS, record with sudo instead. The commands themselves are portable.
#
# Grant the capability before recording. Without it, `scan --fast` falls back to
# a connect scan and prints the fallback notice, so the frame meant to show the
# SYN scanner would show it not being used. (`ping` would likely still work,
# since it falls back to unprivileged ICMP datagram sockets on most Linux
# systems, and `trace` would fail outright — but neither is what this recording
# is for.)
#
# vhs: https://github.com/charmbracelet/vhs

Output docs/demo.gif

Set Shell bash
Set FontSize 15
Set Width 1200
Set Height 780
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms
Set PlaybackSpeed 1.0

# Use the capability-granted build for the recording, and keep the prompt short
# so the commands are what stands out.
Hide
Type "export PATH=/tmp:$PATH; PS1='$ '; clear"  Enter
Show

Sleep 500ms

# ── Ping several hosts at once ───────────────────────────────────────────────
# Two hosts, 3 packets each at a 1s interval, measured at ~2.2s locally. The
# sleep clears the 5s per-run timeout instead, so a slow or unreachable host
# cannot leave the next keystrokes landing mid-command.
Type "netdiag ping google.com cloudflare.com"  Sleep 400ms  Enter
Sleep 6s

Type "clear"  Enter  Sleep 300ms

# ── Half-open SYN scan ───────────────────────────────────────────────────────
Type "netdiag scan 127.0.0.1 -p 1-1024 --fast"  Sleep 400ms  Enter
Sleep 3s

Type "clear"  Enter  Sleep 300ms

# ── Both scan methods, measured against the same target ──────────────────────
Type "netdiag scan 127.0.0.1 -p 1-65535 --benchmark"  Sleep 400ms  Enter
Sleep 5s

Type "clear"  Enter  Sleep 300ms

# ── TLS check, and the JSON contract behind every command ────────────────────
Type "netdiag http https://example.com"  Sleep 400ms  Enter
Sleep 3s

Type "netdiag http https://example.com --json | jq '{status: .http_data.status_code, tls_days: .http_data.tls_days_left}'"  Sleep 400ms  Enter
Sleep 3s

Type "clear"  Enter  Sleep 300ms

# ── Exit codes are the scripting contract ────────────────────────────────────
Type "netdiag dig github.com MX"  Sleep 400ms  Enter
Sleep 3s

Type "echo \"exit: $?\""  Sleep 300ms  Enter
Sleep 2s

Sleep 1s
