# cctop

Monitor your Claude AI token usage in real-time with an htop-inspired terminal interface.

```
cctop - 15:04:05  cost: $12.45  burn rate: 156.30 tokens/min

Tokens  [||||||||||||||||||||||||||                        ] 52.0% (3,640/7,000)
Session [||||||||||||||||||||||||||||||||||||||||||||||    ] 92.0% (24m remaining)

Tokens: 3,640/7,000  Estimate: 16:45  Reset: 15:30  Status: OK
```

## Features

- Real-time token usage monitoring with htop-style progress bars
- Smart burn rate calculation and depletion predictions
- Automatic plan detection (Pro/Max5/Max20)
- Clean, flicker-free terminal interface
- 5-hour rolling session window tracking
- Customizable timezone for display

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
cctop --plan max5         # ~35,000 tokens
cctop --plan max20        # ~140,000 tokens
cctop --plan custom_max   # Auto-detect from history

# Set custom timezone
cctop --timezone US/Eastern
```

### Display Explanation

- **Tokens bar**: Shows current token usage (green → yellow → red)
- **Session bar**: Shows session progress (blue, 0-100% over 5 hours)
- **Status indicators**:
  - `OK` - Tokens will last until session ends
  - `WARNING` - Tokens will run out before session ends
  - `LIMIT EXCEEDED` - Already over token limit

### How Sessions Work

Claude Code uses a 5-hour rolling window system:
- Sessions start with your first message to Claude
- Each session lasts exactly 5 hours from that first message
- Token limits apply within each 5-hour session
- New sessions begin automatically after the previous one ends

## Building

```bash
git clone https://github.com/Sixeight/cctop.git
cd cctop
make build
```

## Development

```bash
make test    # Run tests
make fmt     # Format code
make lint    # Run linter
make help    # Show all commands
```

## Credits

Inspired by [Claude Code Usage Monitor](https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor) and built upon [ccusage](https://github.com/ryoppippi/ccusage).

## License

MIT