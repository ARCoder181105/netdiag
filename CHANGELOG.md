# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- 🏓 Concurrent ping - Test connectivity to multiple hosts simultaneously
- 📡 Speed test - Measure internet download/upload speeds
- 🗺️ Traceroute - Discover network paths to destinations
- 🔍 Port scanner - High-performance concurrent TCP port scanning
- 🌐 HTTP health check - Verify website status and SSL certificates
- 📋 DNS lookup - Query various DNS record types (A, MX, TXT, NS, CNAME)
- 📖 WHOIS lookup - Retrieve domain registration information
- 🔎 Network discovery - Scan local network for active devices

## [0.1.0] - 2026-01-14

### Added
- Initial project setup
- Core CLI structure using Cobra
- Basic command framework

---

## Release Process

1. Update this CHANGELOG.md with all changes since last release
2. Update version in main.go
3. Commit changes: `git commit -am "Release vX.Y.Z"`
4. Create and push tag: `git tag vX.Y.Z && git push origin vX.Y.Z`
5. GitHub Actions will automatically build and publish the release
