package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Display handles all terminal display operations
type Display struct {
	timezone *time.Location
	config   *DisplayConfig
}

// NewDisplay creates a new Display instance
func NewDisplay(timezone string) *Display {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Tokyo")
	}

	return &Display{
		timezone: loc,
	}
}

// Render builds the complete display output for a session
func (d *Display) Render(session *Session, estimator *TokenLimitEstimator, plan string) string {
	var buffer strings.Builder

	// Update display config
	d.config = &DisplayConfig{
		CurrentTime: time.Now(),
		Timezone:    d.timezone,
		BurnRate:    session.BurnRate,
	}

	// Build display sections
	d.renderHeader(&buffer, session)
	d.renderTokenBar(&buffer, session.Metrics.Tokens)
	d.renderTimeBar(&buffer, session.Metrics.Time)
	d.renderStatusBar(&buffer, session)

	// Add notifications
	d.renderNotifications(&buffer, session, plan)

	// Add accuracy warning if needed
	if warning := estimator.GetAccuracyReport(plan, session.Block.TotalTokens, session.Metrics.Tokens.Limit); warning != "" {
		buffer.WriteString("\n" + color.YellowString(warning))
	}

	return buffer.String()
}

// renderHeader renders the header section with model information
func (d *Display) renderHeader(buffer *strings.Builder, session *Session) {
	modelInfo := d.formatModelInfo(session.PrimaryModel, session.CurrentModels)

	fmt.Fprintf(buffer, "cctop - %s  cost: $%.2f  burn rate: %.2f tokens/min  %s\n\n",
		d.config.CurrentTime.Format("15:04:05"),
		session.TodayCost,
		d.config.BurnRate,
		modelInfo)
}

// renderTokenBar renders the token usage progress bar
func (d *Display) renderTokenBar(buffer *strings.Builder, tokens TokenMetrics) {
	fmt.Fprintf(buffer, "Tokens  %s %.1f%% (%s/%s)\n",
		d.createProgressBar(tokens.Percentage, false, config.Plan),
		tokens.Percentage,
		formatNumber(tokens.Used),
		formatNumber(tokens.Limit))
}

// renderTimeBar renders the session time progress bar
func (d *Display) renderTimeBar(buffer *strings.Builder, times TimeMetrics) {
	fmt.Fprintf(buffer, "Session %s %.1f%% (%s remaining)\n\n",
		d.createProgressBar(times.ProgressPercentage, true, ""),
		times.ProgressPercentage,
		formatTime(times.MinutesRemaining))
}

// renderStatusBar renders the status information bar
func (d *Display) renderStatusBar(buffer *strings.Builder, session *Session) {
	predictedEnd := session.GetPredictedEndTime(d.config.CurrentTime)

	fmt.Fprintf(buffer, "Tokens: %s/%s  Estimate: %s  Reset: %s  ",
		formatNumber(session.Metrics.Tokens.Used),
		formatNumber(session.Metrics.Tokens.Limit),
		predictedEnd.In(d.timezone).Format("15:04"),
		session.EndTime.In(d.timezone).Format("15:04"))

	// Status message with color
	status := session.GetStatus()
	switch session.GetStatusColor() {
	case "red":
		buffer.WriteString(color.RedString("Status: %s", status))
	case "yellow":
		buffer.WriteString(color.YellowString("Status: %s", status))
	default:
		buffer.WriteString(color.GreenString("Status: %s", status))
	}
}

// renderNotifications adds any relevant notifications
func (d *Display) renderNotifications(buffer *strings.Builder, session *Session, plan string) {
	if session.Metrics.Tokens.Used > 7000 && plan == "pro" && session.Metrics.Tokens.Limit > 7000 {
		fmt.Fprintf(buffer, "\n%s",
			color.HiBlackString("Note: Auto-switched to auto plan (%s tokens)",
				formatNumber(session.Metrics.Tokens.Limit)))
	}
}

// createProgressBar creates a colored progress bar with optional switch line
func (d *Display) createProgressBar(percentage float64, isTime bool, plan string) string {
	// Ensure percentage is within valid range
	if percentage < 0 {
		percentage = 0
	} else if percentage > 100 {
		percentage = 100
	}

	filled := int(float64(ProgressBarWidth) * percentage / 100)
	filled = clampInt(filled, 0, ProgressBarWidth)

	// Calculate switch line position for Max plans
	var switchLinePos int = -1
	if !isTime && plan != "" {
		switch plan {
		case "max5":
			switchLinePos = int(float64(ProgressBarWidth) * 20 / 100) // 20% for Max5
		case "max20":
			switchLinePos = int(float64(ProgressBarWidth) * 50 / 100) // 50% for Max20
		}
	}

	// Build the bar with switch line marker
	var barParts []string
	for i := 0; i < ProgressBarWidth; i++ {
		if i == switchLinePos {
			barParts = append(barParts, "|") // Switch line marker (will be colored later)
		} else if i < filled {
			barParts = append(barParts, "|")
		} else {
			barParts = append(barParts, " ")
		}
	}

	if isTime {
		// Time bar: color only the filled portion (excluding switch line)
		var coloredParts []string
		for i, part := range barParts {
			if i < filled && part != color.RedString("|") {
				coloredParts = append(coloredParts, color.BlueString(part))
			} else {
				coloredParts = append(coloredParts, part)
			}
		}
		return fmt.Sprintf("[%s]", strings.Join(coloredParts, ""))
	}

	// Token bar colors based on percentage
	var coloredParts []string
	for i, part := range barParts {
		if i < filled && part == "|" {
			// Color the filled portion including switch line
			if i == switchLinePos {
				// Switch line: red if not crossed, same as other bars if crossed
				if percentage <= float64(switchLinePos)*100/float64(ProgressBarWidth) {
					coloredParts = append(coloredParts, color.RedString(part))
				} else {
					// After crossing switch line, use same color as other bars
					switch {
					case percentage < 60:
						coloredParts = append(coloredParts, color.GreenString(part))
					case percentage < 80:
						coloredParts = append(coloredParts, color.YellowString(part))
					default:
						coloredParts = append(coloredParts, color.RedString(part))
					}
				}
			} else {
				// Regular bar coloring
				switch {
				case percentage < 60:
					coloredParts = append(coloredParts, color.GreenString(part))
				case percentage < 80:
					coloredParts = append(coloredParts, color.YellowString(part))
				default:
					coloredParts = append(coloredParts, color.RedString(part))
				}
			}
		} else {
			coloredParts = append(coloredParts, part)
		}
	}

	return fmt.Sprintf("[%s]", strings.Join(coloredParts, ""))
}

// formatModelInfo formats model information for display
func (d *Display) formatModelInfo(primaryModel string, allModels []string) string {
	modelText := fmt.Sprintf("model: %s", primaryModel)

	// Color non-Opus models with light red to indicate they're not the premium model
	if !strings.Contains(strings.ToLower(primaryModel), "opus") {
		return color.HiRedString(modelText)
	}

	// Opus models display without color (default)
	return modelText
}

// RenderError displays an error message
func (d *Display) RenderError(message string) string {
	return message + "\n"
}
