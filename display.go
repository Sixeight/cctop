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
	d.renderHeader(&buffer, session.TodayCost)
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

// renderHeader renders the header section
func (d *Display) renderHeader(buffer *strings.Builder, todayCost float64) {
	fmt.Fprintf(buffer, "cctop - %s  cost: $%.2f  burn rate: %.2f tokens/min\n\n",
		d.config.CurrentTime.Format("15:04:05"),
		todayCost,
		d.config.BurnRate)
}

// renderTokenBar renders the token usage progress bar
func (d *Display) renderTokenBar(buffer *strings.Builder, tokens TokenMetrics) {
	fmt.Fprintf(buffer, "Tokens  %s %.1f%% (%s/%s)\n",
		d.createProgressBar(tokens.Percentage, false),
		tokens.Percentage,
		formatNumber(tokens.Used),
		formatNumber(tokens.Limit))
}

// renderTimeBar renders the session time progress bar
func (d *Display) renderTimeBar(buffer *strings.Builder, times TimeMetrics) {
	fmt.Fprintf(buffer, "Session %s %.1f%% (%s remaining)\n\n",
		d.createProgressBar(times.ProgressPercentage, true),
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

// createProgressBar creates a colored progress bar
func (d *Display) createProgressBar(percentage float64, isTime bool) string {
	// Ensure percentage is within valid range
	if percentage < 0 {
		percentage = 0
	} else if percentage > 100 {
		percentage = 100
	}

	filled := int(float64(ProgressBarWidth) * percentage / 100)
	filled = clampInt(filled, 0, ProgressBarWidth)

	bar := strings.Repeat("|", filled) + strings.Repeat(" ", ProgressBarWidth-filled)

	if isTime {
		return fmt.Sprintf("[%s]", color.BlueString(bar[:filled])+bar[filled:])
	}

	// Token bar colors based on percentage
	var coloredBar string
	switch {
	case percentage < 60:
		coloredBar = color.GreenString(bar[:filled])
	case percentage < 80:
		coloredBar = color.YellowString(bar[:filled])
	default:
		coloredBar = color.RedString(bar[:filled])
	}

	return fmt.Sprintf("[%s]", coloredBar+bar[filled:])
}

// RenderError displays an error message
func (d *Display) RenderError(message string) string {
	return message + "\n"
}
