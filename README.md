# cctop

Monitor your Claude Code token usage in real-time with an htop-inspired terminal interface.

```
cctop - 15:04:05  cost: $12.45  burn rate: 156.30 tokens/min

Tokens  [||||||||||||||||||||||||||                        ] 52.0% (3,640/7,000)
Session [||||||||||||||||||||||||||||||||||||||||||||||    ] 92.0% (24m remaining)

Tokens: 3,640/7,000 (pro)  Estimate: 16:45  Reset: 15:30  Status: OK
```

## Features

- Real-time token usage monitoring with plan detection
- Auto mode: Automatically detects your plan level from usage history
- Dynamic token limit estimation that improves over time
- Burn rate calculation with accurate depletion predictions

## Installation

```bash
# Install via go
go install github.com/Sixeight/cctop@latest

# Prerequisites
npm install -g ccusage
```

## Usage

```bash
# Start monitoring (auto-detects your plan)
cctop

# Override with specific plan
cctop --plan pro          # Force Pro plan limits
cctop --plan max5         # Force Max5 plan limits
cctop --plan max20        # Force Max20 plan limits

# Custom timezone
cctop --timezone US/Eastern
```

### Display Explanation

- **Tokens bar**: Shows current token usage (green → yellow → red)
- **Session bar**: Shows session progress (blue, 0-100% over 5 hours)
- **Plan indicator**: Shows current plan in footer (auto mode displays detected plan)
- **Status indicators**:
  - `OK` - Tokens will last until session ends
  - `WARNING` - Tokens will run out before session ends
  - `LIMIT EXCEEDED` - Already over token limit
- **Accuracy warning**: Shows when token limit estimation may be inaccurate

### How It Works

- Sessions last 5 hours from first message
- Token limits reset with each new session
- Auto mode detects your plan from usage history (100k+ → Max20, 25k+ → Max5)
- Estimates improve with more usage data

## Credits

Inspired by [Claude Code Usage Monitor](https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor) and built upon [ccusage](https://github.com/ryoppippi/ccusage).

## License

MIT

