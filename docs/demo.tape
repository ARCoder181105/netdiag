# netdiag demo recording.
#
# Produces docs/demo.gif, which the README references.
#
#   go build -o /tmp/netdiag . && sudo setcap cap_net_raw+ep /tmp/netdiag
#   vhs docs/demo.tape
#
# The setcap step matters: without it, ping and trace fail and `scan --fast`
# quietly falls back to the connect scan, so the recording would show the
# fallback notice rather than the feature.
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
Type "netdiag ping google.com cloudflare.com"  Sleep 400ms  Enter
Sleep 4s

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
