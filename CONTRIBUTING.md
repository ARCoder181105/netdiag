# Contributing to netdiag 🤝

Thank you for your interest in contributing to **netdiag**! We welcome contributions from the community, whether it's bug fixes, new features, documentation improvements, or suggestions.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Submitting Changes](#submitting-changes)
- [Feature Requests](#feature-requests)
- [Reporting Bugs](#reporting-bugs)
- [Community](#community)

## 📜 Code of Conduct

We are committed to providing a welcoming and inclusive environment for all contributors. Please:

- Be respectful and considerate in all interactions
- Accept constructive criticism gracefully
- Focus on what's best for the project and community
- Show empathy towards other community members

## 🚀 Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.25 or higher** - [Download Go](https://go.dev/dl/)
- **Git** - [Install Git](https://git-scm.com/downloads)
- **Administrator/root privileges** (for testing ICMP operations)

### Fork and Clone

1. **Fork the repository** on GitHub by clicking the "Fork" button
2. **Clone your fork** locally:

```bash
git clone https://github.com/YOUR_USERNAME/netdiag.git
cd netdiag
```

3. **Add upstream remote** to keep your fork in sync: 

```bash
git remote add upstream https://github.com/ARCoder181105/netdiag. git
```

## 🛠️ Development Setup

### Install Dependencies

```bash
# Download all Go module dependencies
go mod download

# Verify dependencies
go mod verify
```

### Build the Project

```bash
# Build the binary
go build -o netdiag

# On Linux, grant ICMP capabilities for testing
sudo setcap cap_net_raw+ep ./netdiag
```

### Run the Application

```bash
# Test the build
./netdiag --help

# Test a specific command
./netdiag ping google.com
```

### Development Build with Hot Reload

For active development, you can use `go run`:

```bash
go run main.go ping google.com
```

## 📁 Project Structure

Understanding the project layout will help you navigate the codebase:

```
netdiag/
├── main.go                 # Application entry point
├── cmd/                    # Command definitions (Cobra commands)
│   ├── ping. go            # Ping command
│   ├── speedtest.go       # Speed test command
│   ├── trace.go           # Traceroute command
│   ├── scan.go            # Port scanner command
│   ├── http.go            # HTTP health check command
│   ├── dig.go             # DNS lookup command
│   ├── whois.go           # WHOIS lookup command
│   └── discover.go        # Network discovery command
├── pkg/                    # Reusable packages and utilities
│   └── [utility modules]
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── LICENSE                 # License file
├── Readme.md              # Project documentation
└── CONTRIBUTING.md        # This file
```

### Key Components

- **`cmd/`**: Each file represents a subcommand using the Cobra framework
- **`pkg/`**: Shared utilities, helpers, and business logic
- **`main.go`**: Bootstraps the CLI and initializes Cobra

## 🤝 How to Contribute

### 1. Pick an Issue or Feature

- Browse [open issues](https://github.com/ARCoder181105/netdiag/issues)
- Look for issues labeled `good first issue` or `help wanted`
- Comment on the issue to let others know you're working on it

### 2. Create a Feature Branch

```bash
# Update your local main branch
git checkout main
git pull upstream main

# Create a new feature branch
git checkout -b feature/your-feature-name
```

**Branch Naming Conventions:**
- `feature/feature-name` - New features
- `fix/bug-description` - Bug fixes
- `docs/update-description` - Documentation updates
- `refactor/description` - Code refactoring

### 3. Make Your Changes

- Write clean, readable code following [Go best practices](https://go.dev/doc/effective_go)
- Keep commits atomic and focused
- Write meaningful commit messages (see [Commit Guidelines](#commit-message-guidelines))

### 4. Test Your Changes

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./... 

# Run tests with coverage
go test -cover ./... 

# Test the specific command you modified
./netdiag [your-command] [arguments]
```

### 5. Commit and Push

```bash
# Stage your changes
git add .

# Commit with a descriptive message
git commit -m "feat: add IPv6 support to ping command"

# Push to your fork
git push origin feature/your-feature-name
```

## 📝 Coding Standards

### Go Code Style

- Follow the [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` to format your code: 
  ```bash
  gofmt -w . 
  ```
- Use `golint` or `golangci-lint` for linting:
  ```bash
  golangci-lint run
  ```

### Best Practices

1. **Error Handling**: Always handle errors explicitly
   ```go
   result, err := someFunction()
   if err != nil {
       return fmt.Errorf("failed to execute: %w", err)
   }
   ```

2. **Concurrency**: Use `errgroup` for concurrent operations
   ```go
   group, ctx := errgroup.WithContext(context.Background())
   group.SetLimit(20) // Limit concurrent operations
   ```

3. **Naming Conventions**:
   - Use camelCase for local variables:  `hostName`, `portNumber`
   - Use PascalCase for exported functions:  `ExecutePing()`, `ScanPorts()`
   - Use descriptive names:  `maxHops` instead of `m`

4. **Comments**:
   - Add comments for exported functions and complex logic
   ```go
   // ScanPorts performs a concurrent TCP port scan on the specified host. 
   // It returns a slice of open ports or an error if the scan fails.
   func ScanPorts(host string, ports []int) ([]int, error) {
       // Implementation
   }
   ```

5. **Constants**: Define magic numbers as constants
   ```go
   const (
       DefaultTimeout = 5 * time.Second
       MaxConcurrentConnections = 100
   )
   ```

### Commit Message Guidelines

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, no logic change)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(scan): add UDP port scanning support

Implemented UDP scanning alongside existing TCP scanning. 
Uses timeout-based response detection for closed ports. 

Closes #42
```

```
fix(ping): resolve packet loss calculation error

Fixed integer overflow when calculating packet loss percentage
for high packet counts. 
```

## 🧪 Testing Guidelines

### Writing Tests

- Place test files alongside source files:  `ping. go` → `ping_test.go`
- Use table-driven tests for multiple test cases: 

```go
func TestParsePortRange(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []int
        wantErr  bool
    }{
        {"Single port", "80", []int{80}, false},
        {"Port range", "80-82", []int{80, 81, 82}, false},
        {"Invalid range", "abc", nil, true},
    }
    
    for _, tt := range tests {
        t.Run(tt. name, func(t *testing. T) {
            result, err := ParsePortRange(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParsePortRange() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if ! reflect.DeepEqual(result, tt.expected) {
                t.Errorf("ParsePortRange() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./... 

# Run tests with coverage
go test -cover ./...  -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

## 📬 Submitting Changes

### Pull Request Process

1. **Update your branch** with the latest upstream changes: 
   ```bash
   git checkout main
   git pull upstream main
   git checkout feature/your-feature-name
   git rebase main
   ```

2. **Push your changes** to your fork:
   ```bash
   git push origin feature/your-feature-name --force-with-lease
   ```

3. **Create a Pull Request**:
   - Go to the [netdiag repository](https://github.com/ARCoder181105/netdiag)
   - Click "New Pull Request"
   - Select your fork and branch
   - Fill out the PR template (see below)

### Pull Request Template

```markdown
## Description
<!-- Provide a clear description of what this PR does -->

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Related Issue
<!-- Link to the issue this PR addresses -->
Fixes #(issue number)

## Testing
<!-- Describe how you tested your changes -->
- [ ] Tested on Linux
- [ ] Tested on macOS
- [ ] Tested on Windows
- [ ] Added unit tests
- [ ] All existing tests pass

## Checklist
- [ ] My code follows the project's coding standards
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have updated the documentation accordingly
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix/feature works
```

### Review Process

- Maintainers will review your PR within a few days
- Address any feedback or requested changes
- Once approved, a maintainer will merge your PR

## 💡 Feature Requests

We love new ideas! Here are some features we'd like to see:

- [ ] **MTR (My Traceroute)** - Continuous latency monitoring
- [ ] **IP Geolocation** - Geographic location lookup for IP addresses
- [ ] **mDNS/Zeroconf** - Service discovery on local networks
- [ ] **JSON Output Mode** - Machine-readable output (`--json` flag)
- [ ] **Configuration File** - YAML/TOML config file support
- [ ] **IPv6 Support** - Full IPv6 support for all commands
- [ ] **Packet Capture** - Save ICMP packets to PCAP files
- [ ] **Continuous Monitoring** - Watch mode for ongoing diagnostics
- [ ] **Custom DNS Servers** - Specify DNS resolver for queries
- [ ] **Bandwidth Monitoring** - Real-time network usage tracking

To propose a new feature:

1. [Open an issue](https://github.com/ARCoder181105/netdiag/issues/new)
2. Use the label `enhancement`
3. Describe the feature, use cases, and potential implementation

## 🐛 Reporting Bugs

Found a bug? Help us fix it! 

### Before Reporting

- Check [existing issues](https://github.com/ARCoder181105/netdiag/issues) to avoid duplicates
- Verify the bug exists on the latest version

### Bug Report Template

```markdown
## Bug Description
<!-- A clear and concise description of the bug -->

## Steps to Reproduce
1. Run command: `netdiag ... `
2. Observe the error:  ... 

## Expected Behavior
<!-- What you expected to happen -->

## Actual Behavior
<!-- What actually happened -->

## Environment
- OS: [e.g., Ubuntu 22.04, macOS 13, Windows 11]
- Go Version: [run `go version`]
- netdiag Version: [run `netdiag --version` if available]

## Additional Context
<!-- Screenshots, logs, or other relevant information -->
```

## 🌐 Community

- **Issues**: [GitHub Issues](https://github.com/ARCoder181105/netdiag/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ARCoder181105/netdiag/discussions) (if enabled)
- **Maintainer**: [@ARCoder181105](https://github.com/ARCoder181105)

## 📄 License

By contributing to netdiag, you agree that your contributions will be licensed under the same license as the project (see [LICENSE](LICENSE)).

---

**Thank you for contributing to netdiag!  🎉**