# Security Policy

## Supported Versions

Security fixes are applied to the latest released minor version. Older versions
are not backported.

| Version | Supported |
| ------- | --------- |
| 0.3.x   | ✅        |
| < 0.3   | ❌        |

## Reporting a Vulnerability

Please **do not** open a public issue for security problems.

Report privately through
[GitHub Security Advisories](https://github.com/ARCoder181105/netdiag/security/advisories/new),
or email **adityarana181105@gmail.com**.

Include:

- A description of the issue and its impact
- Steps to reproduce, or a proof of concept
- The netdiag version (`netdiag --version`) and your OS

You can expect an acknowledgement within 7 days and a status update within 30
days. If a fix is released, you will be credited in the changelog unless you
ask otherwise.

## Scope

netdiag is a diagnostic client. Issues most relevant to it include:

- Remote input (DNS, WHOIS, HTTP, ICMP responses) causing a crash, hang, or
  memory exhaustion
- Certificate validation being bypassed when `--skip-tls` was **not** passed
- Log or config file handling that discloses credentials or writes outside the
  intended path

Out of scope:

- The behavior of `--skip-tls`. It disables certificate verification by
  design, prints a warning, and is flagged in JSON output as
  `tls_verify_skipped`.
- Needing `CAP_NET_RAW` or root for ICMP operations. That is an OS requirement.
- Using netdiag to scan hosts you do not have permission to test. See
  "Responsible use" in the README.

## Elevated Privileges

`netdiag trace` requires raw socket access, and `netdiag ping` / `netdiag
discover` require either raw sockets or unprivileged ICMP datagram sockets.

Prefer granting only the needed capability over running the whole binary as
root:

```bash
sudo setcap cap_net_raw+ep /usr/local/bin/netdiag
```
