# cctop

Monitor your Claude Code token usage in real-time with an htop-inspired terminal interface.

```
cctop - 15:04:05  cost: $12.45  burn rate: 156.30 tokens/min

Tokens  [||||||||||||||||||||||||||                        ] 52.0% (3,640/7,000)
Session [||||||||||||||||||||||||||||||||||||||||||||||    ] 92.0% (24m remaining)

Tokens: 3,640/7,000 (pro)  Estimate: 16:45  Reset: 15:30  Status: OK
123 tokens/msg (136,759 tokens, 446 msgs) x 45 messages (p40)
https://support.anthropic.com/en/articles/11014257-about-claude-s-max-plan-usage
```

## Features

- Real-time token usage monitoring with plan detection
- Auto mode: Automatically detects your plan level from usage history
- Dynamic token limit estimation based on actual message data
- Multiple estimation methods (percentiles, trimmed mean, mode, average)
- Burn rate calculation with accurate depletion predictions
- Shows estimation reasoning with token usage details

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

# Custom estimation method
cctop --est p25           # Use 25th percentile (conservative)
cctop --est median        # Use median
cctop --est trim10        # Use 10% trimmed mean

# List available estimation methods
cctop list-est
```

### Display Explanation

- **Tokens bar**: Shows current token usage (green → yellow → red)
- **Session bar**: Shows session progress (blue, 0-100% over 5 hours)
- **Plan indicator**: Shows current plan in footer (auto mode displays detected plan)
- **Status indicators**:
  - `OK` - Tokens will last until session ends
  - `WARNING` - Tokens will run out before session ends
  - `LIMIT EXCEEDED` - Already over token limit
- **Estimation info**: Shows how token limit was calculated
  - Format: `123 tokens/msg (136,759 tokens, 446 msgs) x 45 messages (p40)`
  - Shows: tokens per message, total tokens/messages from highest session, plan message limit, and estimation method

### How It Works

- Sessions last 5 hours from first message
- Token limits reset with each new session
- Auto mode detects your plan from usage history (100k+ → Max20, 25k+ → Max5)
- Estimation reads actual message token data from Claude's JSONL logs
- Default estimation uses 40th percentile (conservative but realistic)

### Estimation Methods

- **Percentile-based**: `pNN` where NN is 1-99 (e.g., `p25`, `p40`, `p90`)
- **Trimmed mean**: `trimNN` where NN is 0-49 (e.g., `trim10`, `trim20`)
- **Other methods**: `median` (same as p50), `mode`, `avg`
- Default: `p40` (40th percentile)

## Credits

Inspired by [Claude Code Usage Monitor](https://github.com/Maciek-roboblog/Claude-Code-Usage-Monitor) and built upon [ccusage](https://github.com/ryoppippi/ccusage).

## License

MIT

