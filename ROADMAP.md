# Roadmap

This document outlines the planned features and improvements for netdiag.

## Current Version: 0.1.0

Initial release with core network diagnostic features.

## v0.2.0 - Stability & Usability (Planned)

**Focus**: Improve reliability, error handling, and user experience

### Features
- [ ] Comprehensive unit tests for all commands
- [ ] `--json` flag for machine-readable output
- [ ] `--version` flag to show version information
- [ ] Improved error messages with actionable suggestions
- [ ] Progress bars for long-running operations (scan, discover)
- [ ] Timeout handling improvements
- [ ] Color output detection (disable on non-TTY)

### Developer Experience
- [ ] CI/CD pipeline with automated releases
- [ ] Code coverage reports
- [ ] Contributor documentation

## v0.3.0 - Advanced Features (Planned)

**Focus**: Add more advanced network diagnostic capabilities

### Features
- [ ] IPv6 support for all commands
- [ ] Configuration file support (`~/.netdiag.yaml`)
- [ ] Custom DNS resolver configuration
- [ ] PCAP export for packet captures
- [ ] Save results to file (`--output` flag)
- [ ] HTTP/2 and HTTP/3 support in `http` command
- [ ] mDNS/Zeroconf service discovery
- [ ] Certificate chain validation in `http` command

### Enhancements
- [ ] Concurrent DNS lookups in `discover`
- [ ] GeoIP location lookup for IP addresses
- [ ] Reverse DNS lookups
- [ ] Network interface selection

## v0.4.0 - Monitoring & Analytics (Planned)

**Focus**: Continuous monitoring and data analysis

### Features
- [ ] MTR (My Traceroute) - Continuous traceroute with statistics
- [ ] Continuous monitoring mode for all commands
- [ ] Historical data tracking
- [ ] Bandwidth monitoring and graphing
- [ ] Packet loss trending
- [ ] Alert thresholds and notifications
- [ ] Export to popular formats (CSV, JSON, InfluxDB)

### UI/UX
- [ ] Interactive TUI mode (using bubbletea)
- [ ] Dashboard view for monitoring
- [ ] Real-time graphs and charts

## v1.0.0 - Production Ready (Future)

**Focus**: Stable, well-tested, production-ready release

### Goals
- [ ] 80%+ test coverage
- [ ] Complete documentation
- [ ] Pre-built binaries for all major platforms
- [ ] Homebrew formula
- [ ] Debian/RPM packages
- [ ] Snap/Flatpak packages
- [ ] Docker image
- [ ] Chocolatey package (Windows)
- [ ] Performance benchmarks
- [ ] Security audit

### Documentation
- [ ] Complete API documentation
- [ ] Video tutorials
- [ ] Use case examples
- [ ] Troubleshooting guide
- [ ] Performance tuning guide

## Community Wishlist

Features requested by the community:

- Network performance profiling
- VPN connection testing
- WebSocket testing
- SMTP/IMAP connectivity testing
- Database connection testing (MySQL, PostgreSQL, Redis)
- Load testing capabilities
- Distributed testing (client-server mode)
- Plugin system for custom checks
- Web UI for remote monitoring
- Mobile apps (iOS/Android)

## Contributing Ideas

Have an idea for netdiag? We'd love to hear it!

1. Check existing issues and roadmap
2. Open a new issue with the `enhancement` label
3. Describe your use case and proposed solution
4. Discuss with maintainers and community
5. Start implementing (or request help!)

---

**Note**: This roadmap is subject to change based on community feedback and priorities.

**Last Updated**: 2026-01-14
