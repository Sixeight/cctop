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

	// Resolve actual plan for display (auto -> detected plan)
	displayPlan := estimator.GetActualPlan(plan, session.AllBlocks)

	// Build display sections
	d.renderHeader(&buffer, session)
	d.renderTokenBar(&buffer, session.Metrics.Tokens)
	d.renderTimeBar(&buffer, session.Metrics.Time)
	d.renderStatusBar(&buffer, session, displayPlan)

	// Add notifications
	d.renderNotifications(&buffer, session, plan)

	// Add estimation info
	d.renderEstimationInfo(&buffer, estimator, session, displayPlan)

	return buffer.String()
}

// renderHeader renders the header section
func (d *Display) renderHeader(buffer *strings.Builder, session *Session) {
	fmt.Fprintf(buffer, "cctop - %s  cost: $%.2f  burn rate: %.2f tokens/min\n\n",
		d.config.CurrentTime.Format("15:04:05"),
		session.TodayCost,
		d.config.BurnRate)
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
func (d *Display) renderStatusBar(buffer *strings.Builder, session *Session, plan string) {
	predictedEnd := session.GetPredictedEndTime(d.config.CurrentTime)

	fmt.Fprintf(buffer, "Tokens: %s/%s (%s)  Estimate: %s  Reset: %s  ",
		formatNumber(session.Metrics.Tokens.Used),
		formatNumber(session.Metrics.Tokens.Limit),
		plan,
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

// renderEstimationInfo shows how the token limit was estimated
func (d *Display) renderEstimationInfo(buffer *strings.Builder, estimator *TokenLimitEstimator, session *Session, displayPlan string) {
	info := estimator.GetEstimationInfo()
	if info.SessionIndex == 0 {
		// No estimation info available
		return
	}

	// Get plan message limit
	var planMessages int
	switch displayPlan {
	case "pro":
		planMessages = ProPlanMessages
	case "max5":
		planMessages = Max5PlanMessages
	case "max20":
		planMessages = Max20PlanMessages
	default:
		planMessages = ProPlanMessages
	}

	// Format: "300 tokens/msg (13000 tokens, 500 msgs) x 45 messages (p40)"
	fmt.Fprintf(buffer, "\n%s",
		color.HiBlackString("%d tokens/msg (%s tokens, %d msgs) x %d messages (%s)",
			info.TokensPerMsg,
			formatNumber(info.TotalTokens),
			info.Messages,
			planMessages,
			estimator.GetEstimationMethod()))

	// Add link to Claude usage documentation
	fmt.Fprintf(buffer, "\n%s",
		color.HiBlackString("https://support.anthropic.com/en/articles/11014257-about-claude-s-max-plan-usage"))
}

// createProgressBar creates a colored progress bar with optional switch line
func (d *Display) createProgressBar(percentage float64, isTime bool, plan string) string {
	percentage = d.clampPercentage(percentage)
	filled := int(float64(ProgressBarWidth) * percentage / 100)
	filled = clampInt(filled, 0, ProgressBarWidth)

	switchLinePos := d.getSwitchLinePosition(plan, isTime)
	barParts := d.buildBarParts(filled, switchLinePos)

	if isTime {
		return d.colorTimeBar(barParts, filled)
	}
	return d.colorTokenBar(barParts, filled, switchLinePos, percentage)
}

// clampPercentage ensures percentage is within valid range
func (d *Display) clampPercentage(percentage float64) float64 {
	if percentage < 0 {
		return 0
	}
	if percentage > 100 {
		return 100
	}
	return percentage
}

// getSwitchLinePosition calculates switch line position for Max plans
func (d *Display) getSwitchLinePosition(plan string, isTime bool) int {
	if isTime || plan == "" {
		return -1
	}
	switch plan {
	case "max5":
		return int(float64(ProgressBarWidth) * 20 / 100) // 20% for Max5
	case "max20":
		return int(float64(ProgressBarWidth) * 50 / 100) // 50% for Max20
	default:
		return -1
	}
}

// buildBarParts builds the bar structure with markers
func (d *Display) buildBarParts(filled, switchLinePos int) []string {
	var barParts []string
	for i := 0; i < ProgressBarWidth; i++ {
		switch {
		case i == switchLinePos:
			barParts = append(barParts, "|") // Switch line marker
		case i < filled:
			barParts = append(barParts, "|")
		default:
			barParts = append(barParts, " ")
		}
	}
	return barParts
}

// colorTimeBar colors the time progress bar
func (d *Display) colorTimeBar(barParts []string, filled int) string {
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

// colorTokenBar colors the token progress bar
func (d *Display) colorTokenBar(barParts []string, filled, switchLinePos int, percentage float64) string {
	var coloredParts []string
	for i, part := range barParts {
		if i < filled && part == "|" {
			coloredParts = append(coloredParts, d.getTokenBarColor(i, switchLinePos, percentage))
		} else {
			coloredParts = append(coloredParts, part)
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(coloredParts, ""))
}

// getTokenBarColor returns the colored bar segment
func (d *Display) getTokenBarColor(position, switchLinePos int, percentage float64) string {
	if position == switchLinePos {
		return d.getSwitchLineColor(switchLinePos, percentage)
	}
	return d.getRegularBarColor(percentage)
}

// getSwitchLineColor returns color for switch line position
func (d *Display) getSwitchLineColor(switchLinePos int, percentage float64) string {
	switchThreshold := float64(switchLinePos) * 100 / float64(ProgressBarWidth)
	if percentage <= switchThreshold {
		return color.RedString("|")
	}
	return d.getRegularBarColor(percentage)
}

// getRegularBarColor returns color based on percentage
func (d *Display) getRegularBarColor(percentage float64) string {
	switch {
	case percentage < 60:
		return color.GreenString("|")
	case percentage < 80:
		return color.YellowString("|")
	default:
		return color.RedString("|")
	}
}

// RenderError displays an error message
func (d *Display) RenderError(message string) string {
	return message + "\n"
}
