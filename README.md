# cctop

Monitor your Claude Code token usage in real-time with an htop-inspired terminal interface.

```
cctop - 15:04:05  cost: $12.45  burn rate: 156.30 tokens/min

Tokens  [||||||||||||||||||||||||||                        ] 52.0% (3,640/7,000)
Session [||||||||||||||||||||||||||||||||||||||||||||||    ] 92.0% (24m remaining)

Tokens: 3,640/7,000  Estimate: 16:45  Reset: 15:30  Status: OK
```

## Features

- Real-time token usage monitoring
- Dynamic token limit estimation from usage patterns
- Burn rate calculation with depletion predictions
- Automatic plan detection (Pro/Max5/Max20)

## Installation

```bash
# Install via go
go install github.com/Sixeight/cctop@latest

# Prerequisites
npm install -g ccusage
```

## Usage

```bash
# Start monitoring with default settings
cctop

# Use a specific plan
cctop --plan pro          # Dynamic estimation based on Pro plan
cctop --plan max5         # Dynamic estimation based on Max5 plan
cctop --plan max20        # Dynamic estimation based on Max20 plan
cctop --plan custom_max   # Auto-detect from history

# Set custom timezone
cctop --timezone US/Eastern

# Analyze estimation accuracy
cctop analyze
```

### Display Explanation

- **Tokens bar**: Shows current token usage (green → yellow → red)
- **Session bar**: Shows session progress (blue, 0-100% over 5 hours)
- **Status indicators**:
  - `OK` - Tokens will last until session ends
  - `WARNING` - Tokens will run out before session ends
  - `LIMIT EXCEEDED` - Already over token limit
- **Accuracy warning**: Shows when token limit estimation may be inaccurate

### How Sessions Work

Claude Code uses a 5-hour rolling window system:

- Sessions start with your first message to Claude
- Each session lasts exactly 5 hours from that first message
- Token limits apply within each 5-hour session
- New sessions begin automatically after the previous one ends

### Dynamic Token Limit Estimation

cctop now learns from your usage patterns to provide more accurate token limits:

- **Initial estimates** based on official Anthropic message counts
- **Adaptive learning** improves accuracy as you use Claude more
- **Outlier detection** removes anomalous sessions from calculations
- **Hybrid approach** combines historical data with official limits
- **Accuracy monitoring** warns when estimates may be unreliable

The more you use Claude, the more accurate the estimates become!

## Credits

Inspired by [Claude Code Usage Monitor](https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor) and built upon [ccusage](https://github.com/ryoppippi/ccusage).

## License

MIT

