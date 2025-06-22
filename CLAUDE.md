# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cctop is a real-time terminal monitoring tool for Claude AI token usage with an htop-inspired interface. It displays token consumption, burn rate, and session timing information with dynamic token limit estimation based on historical usage patterns.

## Essential Commands

```bash
# Build and test
make all          # Format, lint, test, and build
make build        # Build the binary
make test         # Run tests
make test-coverage # Run tests with coverage report
make fmt          # Format code with gofumpt and goimports
make lint         # Run comprehensive linting with auto-fix

# CI-specific commands
make fmt-check    # Check formatting without modifying files
make lint-check   # Quick linting for CI (errors only)

# Development
go run .                      # Run without building (uses auto plan)
go run . --plan pro           # Run with specific plan
go run . analyze              # Analyze token limit estimation accuracy
go test -v -run TestName      # Run specific test
go test -v ./... -count=1     # Run all tests without cache

# Clean up
make clean        # Remove build artifacts

# Release (local testing)
goreleaser build --snapshot --clean
```

## Architecture & Key Design Decisions

### Core Architecture

The system consists of several key components:

1. **main.go** (~320 lines): Entry point, application flow, and command handling
   - Cobra CLI integration with analyze command
   - Signal handling for graceful shutdown
   - Main update loop orchestration

2. **display.go** (~240 lines): Terminal rendering and UI components
   - ANSI escape sequence terminal control
   - Progress bar rendering with color coding
   - Header, status bar, and notification display
   - Plan name display in footer format: "Tokens: xxx/xxx (plan)"

3. **session.go** (~270 lines): Session data management and model detection
   - Session lifecycle tracking and metrics calculation
   - Active model detection with intelligent heuristics
   - Time and token metrics computation

4. **estimator.go** (~210 lines): Dynamic token limit estimation
   - Hybrid approach combining official limits with historical data
   - Outlier detection using IQR method
   - Adaptive weighting based on sample size and variance
   - Auto plan resolution (auto → detected plan)

5. **config.go** (~95 lines): Application configuration management
   - Plan validation and token limit mappings
   - Threshold configuration for auto-switching
   - Progress bar color configuration

6. **analyze_accuracy.go** (~185 lines): Diagnostic tool for estimation accuracy
   - Temporary/debug tool - excluded from linting in .golangci.yml

### Data Flow

```
ccusage (npm) → JSON → main.go → session.go → display.go
                         ↓           ↓
                   estimator.go → config.go
                         ↓
                 analyze_accuracy.go (diagnostic)
```

### Session Tracking System
The tool implements Claude's 5-hour rolling window session system:
- Sessions start with the first message to Claude (tracked via ccusage)
- Each session lasts exactly 5 hours from start time
- Token limits apply within each 5-hour window
- NOT fixed schedule resets (no 4:00, 9:00, 14:00, etc.)

### Core Dependencies
- `ccusage` npm package: Required for fetching token data (`npm install -g ccusage`)
- The entire tool depends on ccusage JSON output format with `blocks` array containing session data
- JSON structure: `blocks[]` with fields: `startTime`, `actualEndTime`, `totalTokens`, `entries`, `isActive`, `isGap`

### Display Architecture
1. **Update Loop**: 3-second refresh cycle with flicker-free updates
2. **Buffer Strategy**: Build entire display in memory, then single print
3. **ANSI Control**: Uses escape sequences for cursor positioning (no full clear)
4. **Progress Bars**: Custom implementation with color coding:
   - Tokens: green (<60%) → yellow (60-80%) → red (>80%)
   - Session: always blue (neutral time indicator)
5. **Plan Display**: Footer shows current plan in format "Tokens: xxx/xxx (plan)"
   - For auto plan: shows detected plan (pro/max5/max20) without "(auto)" suffix
   - Plan detection based on historical usage patterns

### Token Limit Estimation System

The `TokenLimitEstimator` provides dynamic, learning-based token limit estimation:

1. **Hybrid Approach**: Combines official Anthropic message counts with historical usage data
   - Official limits: Pro=45 msgs, Max5=225 msgs, Max20=900 msgs (×150 tokens/msg default)
   - Historical data: 90th percentile of past sessions after outlier removal (changed from 95th)

2. **Adaptive Weighting**: Trust in historical data varies based on:
   - Sample size: <10 sessions = 30% weight, 20+ sessions = up to 80% weight
   - Data variance: High coefficient of variation reduces trust

3. **Outlier Removal**: Uses IQR (Interquartile Range) method to exclude extreme values
   - Values outside 1.5 * IQR are removed

4. **Accuracy Monitoring**: Warns when estimation deviates >10% from actual usage

### Burn Rate Calculation
- Analyzes all session blocks from the last hour
- Handles overlapping sessions correctly
- Returns tokens/minute rate for predictive estimates

### Plan Detection
- `auto` (default): Automatically detects plan level from usage history
  - History shows 100k+ tokens → Uses Max20 plan (900 messages)
  - History shows 25k+ tokens → Uses Max5 plan (225 messages)
  - Otherwise → Uses Pro plan (45 messages)
- Automatic plan switching when limits are exceeded (e.g., pro → auto at >7000 tokens)
- All plans use dynamic estimation that improves with usage data

## Important Implementation Notes

1. **Time Calculations**: All session timing is based on the active block's StartTime + 5 hours
2. **Error Handling**: ccusage failures result in retry (no crash)
3. **Signal Handling**: Ctrl+C properly restores cursor visibility
4. **Number Formatting**: Custom comma insertion for readability (e.g., 7,000)
5. **JSON Parsing**: Block structure includes `entries` field for message count
6. **Component Initialization**: Global instances (estimator, display, burnCalc) created in main()
7. **Code Quality**: Functions are kept under 15 cyclomatic complexity for maintainability
   - Complex functions are broken down into smaller, focused helper functions
   - Named return values used where appropriate for clarity
8. **Tool Management**: Development tools are managed via go.mod's `tool` directive (Go 1.24+)
   - No manual installation needed
   - Tools run via `go run` commands in Makefile

## Testing Strategy

- Unit tests for all estimation functions
- Mock data tests for various usage patterns
- Accuracy analysis tool (`cctop analyze`) for real-world validation
- Test coverage target: >80% for critical paths
- GitHub Actions CI runs tests on Linux, macOS, and Windows

## CI/CD Pipeline

### GitHub Actions Workflows
1. **ci.yml**: Runs on every push and PR
   - Multi-OS testing (Ubuntu, macOS, Windows)
   - Go 1.24.x
   - Format checking, linting, tests, coverage
   - Dependency vulnerability scanning

2. **goreleaser.yml**: Runs on version tags (v*)
   - Automated multi-platform releases
   - Homebrew formula generation
   - Changelog from commit messages (feat:, fix:, docs:, chore:)

### Linting Configuration
See `.golangci.yml` for comprehensive linting rules:
- Basic: errcheck, govet, ineffassign, staticcheck, unused
- Quality: revive, gocritic, prealloc, unconvert
- Security: gosec
- Test files have relaxed rules
- `analyze_accuracy.go` is excluded from all linters

## Session Specification Reference

See `.claude/session.md` for detailed documentation about Claude's session system based on official Anthropic documentation. Key points:
- 5-hour rolling windows, not fixed schedules
- Sessions start with first message
- Token limits reset with new sessions