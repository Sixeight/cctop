# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cctop is a real-time terminal monitoring tool for Claude AI token usage with an htop-inspired interface. It displays token consumption, burn rate, and session timing information.

## Essential Commands

```bash
# Build and test
make all          # Format, lint, test, and build
make build        # Build the binary
make test         # Run tests
make fmt          # Format code
make lint         # Run linter (requires golangci-lint)

# Development
go run .                      # Run without building
go run . --plan max5          # Run with specific plan
go test -v -run TestName      # Run specific test
```

## Architecture & Key Design Decisions

### Session Tracking System
The tool implements Claude's 5-hour rolling window session system:
- Sessions start with the first message to Claude (tracked via ccusage)
- Each session lasts exactly 5 hours from start time
- Token limits apply within each 5-hour window
- NOT fixed schedule resets (no 4:00, 9:00, 14:00, etc.)

### Core Dependencies
- `ccusage` npm package: Required for fetching token data (`npm install -g ccusage`)
- The entire tool depends on ccusage JSON output format

### Display Architecture
1. **Update Loop**: 3-second refresh cycle with flicker-free updates
2. **Buffer Strategy**: Build entire display in memory, then single print
3. **ANSI Control**: Uses escape sequences for cursor positioning (no full clear)
4. **Progress Bars**: Custom implementation with color coding:
   - Tokens: green (<60%) → yellow (60-80%) → red (>80%)
   - Session: always blue (neutral time indicator)

### Burn Rate Calculation
- Analyzes all session blocks from the last hour
- Handles overlapping sessions correctly
- Returns tokens/minute rate

### Plan Detection
- `custom_max`: Scans historical sessions for highest token usage
- Auto-switches from pro to custom_max when limit exceeded
- Hardcoded limits: pro=7000, max5=35000, max20=140000

## Important Implementation Notes

1. **Time Calculations**: All session timing is based on the active block's StartTime + 5 hours
2. **Error Handling**: ccusage failures result in retry (no crash)
3. **Signal Handling**: Ctrl+C properly restores cursor visibility
4. **Number Formatting**: Custom comma insertion for readability

## Session Specification Reference

See `.claude/session.md` for detailed documentation about Claude's session system based on official Anthropic documentation. Key points:
- 5-hour rolling windows, not fixed schedules
- Sessions start with first message
- Token limits reset with new sessions