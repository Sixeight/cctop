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
make fmt          # Format code
make lint         # Run linter (requires golangci-lint)

# Development
go run .                      # Run without building
go run . --plan max5          # Run with specific plan
go run . analyze              # Analyze token limit estimation accuracy
go test -v -run TestName      # Run specific test

# Clean up
make clean        # Remove build artifacts
```

## Architecture & Key Design Decisions

### Core Architecture

The system consists of several key components:

1. **main.go**: Entry point, display loop, and UI rendering
2. **estimator.go**: Dynamic token limit estimation using ML-like approach
3. **analyze_accuracy.go**: Diagnostic tool for estimation accuracy analysis

### Session Tracking System
The tool implements Claude's 5-hour rolling window session system:
- Sessions start with the first message to Claude (tracked via ccusage)
- Each session lasts exactly 5 hours from start time
- Token limits apply within each 5-hour window
- NOT fixed schedule resets (no 4:00, 9:00, 14:00, etc.)

### Core Dependencies
- `ccusage` npm package: Required for fetching token data (`npm install -g ccusage`)
- The entire tool depends on ccusage JSON output format with `blocks` array containing session data

### Display Architecture
1. **Update Loop**: 3-second refresh cycle with flicker-free updates
2. **Buffer Strategy**: Build entire display in memory, then single print
3. **ANSI Control**: Uses escape sequences for cursor positioning (no full clear)
4. **Progress Bars**: Custom implementation with color coding:
   - Tokens: green (<60%) → yellow (60-80%) → red (>80%)
   - Session: always blue (neutral time indicator)

### Token Limit Estimation System

The new `TokenLimitEstimator` provides dynamic, learning-based token limit estimation:

1. **Hybrid Approach**: Combines official Anthropic message counts with historical usage data
   - Official limits: Pro=45 msgs, Max5=225 msgs, Max20=900 msgs (×150 tokens/msg default)
   - Historical data: 95th percentile of past sessions after outlier removal

2. **Adaptive Weighting**: Trust in historical data varies based on:
   - Sample size: <10 sessions = 30% weight, 20+ sessions = up to 80% weight
   - Data variance: High coefficient of variation reduces trust

3. **Outlier Removal**: Uses IQR (Interquartile Range) method to exclude extreme values

4. **Accuracy Monitoring**: Warns when estimation deviates >10% from actual usage

### Burn Rate Calculation
- Analyzes all session blocks from the last hour
- Handles overlapping sessions correctly
- Returns tokens/minute rate for predictive estimates

### Plan Detection
- `custom_max`: Uses historical data to estimate appropriate limits
- Automatic plan switching when limits are exceeded
- Dynamic limits now replace hardcoded values

## Important Implementation Notes

1. **Time Calculations**: All session timing is based on the active block's StartTime + 5 hours
2. **Error Handling**: ccusage failures result in retry (no crash)
3. **Signal Handling**: Ctrl+C properly restores cursor visibility
4. **Number Formatting**: Custom comma insertion for readability
5. **JSON Parsing**: Block structure includes `entries` field for message count
6. **Estimator Initialization**: Global `estimator` instance created in main()

## Testing Strategy

- Unit tests for all estimation functions
- Mock data tests for various usage patterns
- Accuracy analysis tool (`cctop analyze`) for real-world validation
- Test coverage target: >80% for critical paths

## Session Specification Reference

See `.claude/session.md` for detailed documentation about Claude's session system based on official Anthropic documentation. Key points:
- 5-hour rolling windows, not fixed schedules
- Sessions start with first message
- Token limits reset with new sessions