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
go run . --est p25            # Run with custom estimation method
go run . list-est             # List available estimation methods
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
   - Cobra CLI integration with analyze and list-est commands
   - Signal handling for graceful shutdown
   - Main update loop orchestration

2. **display.go** (~265 lines): Terminal rendering and UI components
   - ANSI escape sequence terminal control
   - Progress bar rendering with color coding
   - Header, status bar, and notification display
   - Estimation info display with method indicator

3. **session.go** (~270 lines): Session data management and model detection
   - Session lifecycle tracking and metrics calculation
   - Active model detection with intelligent heuristics
   - Time and token metrics computation

4. **estimator.go** (~210 lines): Dynamic token limit estimation
   - Hybrid approach combining official limits with historical data
   - Outlier detection using IQR method
   - Adaptive weighting based on sample size and variance
   - Auto plan resolution (auto → detected plan)

5. **estimator_helpers.go** (~85 lines): Helper functions for estimation
   - Separates complex logic to maintain cyclomatic complexity <15
   - Handles custom percentile and trim percentage parsing

6. **jsonl_reader.go** (~230 lines): Reads actual message token data
   - Parses Claude's JSONL log files from ~/.config/claude/projects/
   - Aggregates data across all projects
   - Provides median, trimmed mean, and mode calculations

7. **config.go** (~95 lines): Application configuration management
   - Plan validation and token limit mappings
   - Threshold configuration for auto-switching
   - Progress bar color configuration

8. **analyze_accuracy.go** (~185 lines): Diagnostic tool for estimation accuracy
   - Temporary/debug tool - excluded from linting in .golangci.yml

### Data Flow

```
ccusage (npm) → JSON → main.go → session.go → display.go
                         ↓           ↓
                   estimator.go → config.go
                         ↓
                 jsonl_reader.go
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
5. **Estimation Info**: Shows reasoning in format "123 tokens/msg (136,759 tokens, 446 msgs) x 900 messages (p40)"
   - Displays: tokens/msg, total tokens from highest session, message count, plan limit, estimation method

### Token Limit Estimation System

The `TokenLimitEstimator` provides dynamic, learning-based token limit estimation:

1. **Data Sources**:
   - Primary: Claude's JSONL log files containing actual per-message token counts
   - Fallback: ccusage session totals divided by message count

2. **Estimation Methods** (configurable via --est flag):
   - Percentile-based: `pNN` where NN is 1-99 (default: `p40`)
   - Trimmed mean: `trimNN` where NN is 0-49
   - Other: `median` (alias for p50), `mode`, `avg`

3. **Session Selection**: Always uses the session with highest token consumption

4. **Hybrid Approach**: Combines official Anthropic message counts with historical usage data
   - Official limits: Pro=45 msgs, Max5=225 msgs, Max20=900 msgs
   - Historical data: Selected statistical method applied to message tokens

5. **Adaptive Weighting**: Trust in historical data varies based on:
   - Sample size: <10 sessions = 30% weight, 20+ sessions = up to 80% weight
   - Data variance: High coefficient of variation reduces trust

6. **Outlier Removal**: Uses IQR (Interquartile Range) method to exclude extreme values
   - Values outside 1.5 * IQR are removed

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