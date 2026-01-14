# Developer Guide

Welcome to the netdiag developer guide! This document will help you contribute to the project.

## Quick Start

### Prerequisites

- Go 1.25 or higher
- Git
- golangci-lint (optional, for linting)

### Setup

```bash
# Clone the repository
git clone https://github.com/ARCoder181105/netdiag.git
cd netdiag

# Download dependencies
go mod download

# Build the project
go build -o netdiag

# Run tests
go test ./...
```

### Common Development Tasks

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests with coverage
make test-cover

# Build binary
make build

# Install locally
make install

# Run quick test
make run-ping

# Pre-commit checks (fmt + vet + lint + test)
make pre-commit
```

## Architecture Overview

netdiag is built using a modular command structure pattern:

```
netdiag/
├── main.go           # Entry point
├── cmd/              # Command implementations
│   ├── root.go       # Root command and global flags
│   ├── ping.go       # Ping command
│   ├── scan.go       # Port scanner
│   ├── speedtest.go  # Speed test
│   └── ...           # Other commands
└── pkg/              # Shared packages (if any)
```

### Command Structure

Each command follows this pattern:

1. **Command Definition** - Uses Cobra to define the command, flags, and help text
2. **Input Validation** - Validates user input and flags
3. **Core Logic** - Implements the actual functionality
4. **Output Formatting** - Displays results using tablewriter and color

## How to Add a New Command

Let's walk through adding a new command called `mtr` (continuous traceroute):

### Step 1: Create the command file

Create `cmd/mtr.go`:

```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var mtrCmd = &cobra.Command{
	Use:   "mtr <host>",
	Short: "Continuous traceroute with latency monitoring",
	Long:  `Monitor network path to a host continuously with real-time statistics.`,
	Args:  cobra.ExactArgs(1),
	Run:   runMTR,
}

func init() {
	rootCmd.AddCommand(mtrCmd)
	
	// Add flags
	mtrCmd.Flags().IntP("count", "c", 10, "Number of pings per hop")
	mtrCmd.Flags().IntP("interval", "i", 1, "Seconds between rounds")
}

func runMTR(cmd *cobra.Command, args []string) {
	host := args[0]
	count, _ := cmd.Flags().GetInt("count")
	interval, _ := cmd.Flags().GetInt("interval")
	
	fmt.Printf("Running MTR to %s...\n", host)
	// Implementation here
}
```

### Step 2: Implement the logic

Add your implementation in the `runMTR` function. Use existing patterns from other commands:

- Use `errgroup` for concurrency
- Use `tablewriter` for output
- Use `color` for colored text
- Handle errors gracefully

### Step 3: Test your command

```bash
# Build and test
make build
./netdiag mtr google.com

# Run with different flags
./netdiag mtr google.com -c 20 -i 2
```

### Step 4: Add documentation

Update:
- `README.md` - Add command to the Commands Reference section
- `CHANGELOG.md` - Add to Unreleased section

## Testing

### Unit Tests

Create `cmd/mtr_test.go`:

```go
package cmd

import "testing"

func TestMTRValidation(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{"valid hostname", "google.com", false},
		{"valid IP", "8.8.8.8", false},
		{"empty host", "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test logic here
		})
	}
}
```

Run tests:

```bash
make test
```

### Integration Tests

Integration tests verify end-to-end functionality:

```bash
# Test actual functionality
./netdiag mtr google.com -c 5
```

## Dependencies Management

### Adding a new dependency

```bash
# Add the dependency
go get github.com/example/package

# Tidy modules
make tidy
```

### Updating dependencies

```bash
# Update all dependencies
go get -u ./...

# Update specific dependency
go get -u github.com/example/package

# Verify
make deps
```

## Debugging

### VS Code

The project includes debug configurations in `.vscode/launch.json`:

1. Open VS Code
2. Set breakpoints in your code
3. Press F5 or use Run > Start Debugging
4. Select the appropriate debug configuration

### Command Line (Delve)

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a command
dlv debug -- ping google.com

# Common commands in Delve
(dlv) break main.main     # Set breakpoint
(dlv) continue            # Continue execution
(dlv) next                # Step over
(dlv) step                # Step into
(dlv) print variable      # Print variable value
```

## Release Process

1. **Update version** in `main.go`:
   ```go
   var version = "0.2.0"
   ```

2. **Update CHANGELOG.md** with all changes

3. **Commit changes**:
   ```bash
   git commit -am "Release v0.2.0"
   ```

4. **Create and push tag**:
   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```

5. **GitHub Actions** will automatically:
   - Build binaries for all platforms
   - Generate checksums
   - Create a GitHub release
   - Upload all artifacts

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Run `make fmt` before committing
- Use meaningful variable names
- Add comments for exported functions
- Keep functions small and focused

## Resources

- [Cobra Documentation](https://github.com/spf13/cobra)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Testing](https://go.dev/doc/tutorial/add-a-test)
- [golangci-lint](https://golangci-lint.run/)

## Getting Help

- Open an issue for bugs or feature requests
- Join discussions for questions
- Check existing issues and PRs
