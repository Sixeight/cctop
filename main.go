package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	SessionDurationMinutes = 300.0
	UpdateInterval         = 3 * time.Second
	ProgressBarWidth       = 50
	DefaultTokenLimit      = 7000
)

// Block represents a usage block from ccusage
type Block struct {
	StartTime     string `json:"startTime"`
	ActualEndTime string `json:"actualEndTime"`
	TotalTokens   int    `json:"totalTokens"`
	IsActive      bool   `json:"isActive"`
	IsGap         bool   `json:"isGap"`
}

// CCUsageData represents the JSON response from ccusage
type CCUsageData struct {
	Blocks []Block `json:"blocks"`
}

// DailyUsage represents daily usage data from ccusage
type DailyUsage struct {
	Date      string  `json:"date"`
	TotalCost float64 `json:"totalCost"`
}

// TokenMetrics holds calculated token usage information
type TokenMetrics struct {
	Used       int
	Limit      int
	Percentage float64
	Remaining  int
}

// TimeMetrics holds calculated time information
type TimeMetrics struct {
	SessionEndTime     time.Time
	MinutesRemaining   float64
	ProgressPercentage float64
}

// Display configuration
type DisplayConfig struct {
	CurrentTime time.Time
	BurnRate    float64
	Timezone    *time.Location
}

var (
	plan     string
	timezone string
)

var rootCmd = &cobra.Command{
	Use:   "cctop",
	Short: "Claude Code Usage Monitor - Real-time token usage monitoring",
	Long:  `A beautiful real-time terminal monitoring tool for Claude AI token usage.`,
	Run:   runMonitor,
}

func init() {
	rootCmd.Flags().StringVar(&plan, "plan", "pro", "Claude plan type (pro, max5, max20, custom_max)")
	rootCmd.Flags().StringVar(&timezone, "timezone", "Asia/Tokyo", "Timezone for display")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Terminal control functions
func hideCursor()   { fmt.Print("\033[?25l") }
func showCursor()   { fmt.Print("\033[?25h") }
func clearScreen()  { fmt.Print("\033[2J\033[H") }
func clearAndHome() { fmt.Print("\033[H\033[J") }

func runMonitor(cmd *cobra.Command, args []string) {
	hideCursor()
	defer showCursor()

	setupSignalHandler()
	tokenLimit := getInitialTokenLimit()
	clearScreen()

	for {
		if err := updateDisplay(&tokenLimit); err != nil {
			displayError(err.Error())
			time.Sleep(UpdateInterval)
			continue
		}
		time.Sleep(UpdateInterval)
	}
}

func setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		showCursor()
		fmt.Print("\n")
		os.Exit(0)
	}()
}

func updateDisplay(tokenLimit *int) error {
	usageData := fetchUsageData()
	if usageData == nil {
		return fmt.Errorf("Failed to get usage data")
	}

	activeBlock := findActiveBlock(usageData.Blocks)
	if activeBlock == nil {
		return fmt.Errorf("No active session found")
	}

	display := buildDisplay(activeBlock, usageData.Blocks, tokenLimit)
	clearAndHome()
	fmt.Print(display)
	return nil
}

func displayError(message string) {
	var buffer strings.Builder
	buffer.WriteString(message + "\n")
	clearAndHome()
	fmt.Print(buffer.String())
}

func fetchUsageData() *CCUsageData {
	cmd := exec.Command("ccusage", "blocks", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var data CCUsageData
	if err := json.Unmarshal(output, &data); err != nil {
		return nil
	}

	return &data
}

func findActiveBlock(blocks []Block) *Block {
	for i := range blocks {
		if blocks[i].IsActive {
			return &blocks[i]
		}
	}
	return nil
}

func getInitialTokenLimit() int {
	if plan == "custom_max" {
		data := fetchUsageData()
		if data != nil {
			return getTokenLimit(plan, data.Blocks)
		}
	}
	return getTokenLimit(plan, nil)
}

func getTokenLimit(planType string, blocks []Block) int {
	if planType == "custom_max" && blocks != nil {
		maxTokens := 0
		for _, block := range blocks {
			if !block.IsGap && !block.IsActive && block.TotalTokens > maxTokens {
				maxTokens = block.TotalTokens
			}
		}
		if maxTokens > 0 {
			return maxTokens
		}
	}

	limits := map[string]int{
		"pro":   7000,
		"max5":  35000,
		"max20": 140000,
	}

	if limit, ok := limits[planType]; ok {
		return limit
	}
	return DefaultTokenLimit
}

func buildDisplay(activeBlock *Block, allBlocks []Block, tokenLimit *int) string {
	var buffer strings.Builder

	// Calculate metrics
	tokens := calculateTokenMetrics(activeBlock, allBlocks, tokenLimit)
	display := createDisplayConfig(allBlocks)
	times := calculateTimeMetrics(activeBlock, display.CurrentTime)
	predictedEnd := calculatePredictedEnd(tokens, display.BurnRate, display.CurrentTime, times.SessionEndTime)
	todayTotalCost := fetchTodayTotalCost(display.CurrentTime)

	// Build display sections
	buildHeader(&buffer, display, todayTotalCost)
	buildTokenBar(&buffer, tokens)
	buildTimeBar(&buffer, times)
	buildStatusBar(&buffer, tokens, times, predictedEnd, display)

	// Add notifications if needed
	addNotifications(&buffer, tokens, tokenLimit)

	return buffer.String()
}

func calculateTokenMetrics(activeBlock *Block, allBlocks []Block, tokenLimit *int) TokenMetrics {
	tokensUsed := activeBlock.TotalTokens

	// Auto-switch to custom_max if needed
	if tokensUsed > *tokenLimit && plan == "pro" {
		newLimit := getTokenLimit("custom_max", allBlocks)
		if newLimit > *tokenLimit {
			*tokenLimit = newLimit
		}
	}

	return TokenMetrics{
		Used:       tokensUsed,
		Limit:      *tokenLimit,
		Percentage: float64(tokensUsed) / float64(*tokenLimit) * 100,
		Remaining:  *tokenLimit - tokensUsed,
	}
}

func createDisplayConfig(allBlocks []Block) DisplayConfig {
	currentTime := time.Now()
	return DisplayConfig{
		CurrentTime: currentTime,
		BurnRate:    calculateHourlyBurnRate(allBlocks, currentTime),
		Timezone:    getTimezone(),
	}
}

func calculateTimeMetrics(activeBlock *Block, currentTime time.Time) TimeMetrics {
	// Parse session start time
	startTime, _ := time.Parse(time.RFC3339, activeBlock.StartTime)

	// Session ends exactly 5 hours after start
	sessionEndTime := startTime.Add(5 * time.Hour)
	elapsedMinutes := currentTime.Sub(startTime).Minutes()
	remainingMinutes := sessionEndTime.Sub(currentTime).Minutes()
	if remainingMinutes < 0 {
		remainingMinutes = 0
	}

	// Progress percentage (0% at start, 100% at 5 hours)
	progressPercentage := (elapsedMinutes / SessionDurationMinutes) * 100
	if progressPercentage < 0 {
		progressPercentage = 0
	} else if progressPercentage > 100 {
		progressPercentage = 100
	}

	return TimeMetrics{
		SessionEndTime:     sessionEndTime,
		MinutesRemaining:   remainingMinutes,
		ProgressPercentage: progressPercentage,
	}
}

func calculatePredictedEnd(tokens TokenMetrics, burnRate float64, currentTime, sessionEndTime time.Time) time.Time {
	if burnRate > 0 && tokens.Remaining > 0 {
		minutesToDepletion := float64(tokens.Remaining) / burnRate
		return currentTime.Add(time.Duration(minutesToDepletion) * time.Minute)
	}
	return sessionEndTime
}

func fetchTodayTotalCost(currentTime time.Time) float64 {
	// Get today's date in YYYY-MM-DD format
	todayStr := currentTime.Format("2006-01-02")

	// Run ccusage daily command
	cmd := exec.Command("ccusage", "daily", "--json")
	output, err := cmd.Output()
	if err != nil {
		return 0.0
	}

	// Parse JSON response
	var response struct {
		Daily []DailyUsage `json:"daily"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return 0.0
	}

	// Find today's entry
	for _, day := range response.Daily {
		if day.Date == todayStr {
			return day.TotalCost
		}
	}

	return 0.0
}

func buildHeader(buffer *strings.Builder, config DisplayConfig, todayTotalCost float64) {
	buffer.WriteString(fmt.Sprintf("cctop - %s  cost: $%.2f  burn rate: %.2f tokens/min\n\n",
		config.CurrentTime.Format("15:04:05"),
		todayTotalCost,
		config.BurnRate))
}

func buildTokenBar(buffer *strings.Builder, tokens TokenMetrics) {
	buffer.WriteString(fmt.Sprintf("Tokens  %s %.1f%% (%s/%s)\n",
		createProgressBar(tokens.Percentage, false),
		tokens.Percentage,
		formatNumber(tokens.Used),
		formatNumber(tokens.Limit)))
}

func buildTimeBar(buffer *strings.Builder, times TimeMetrics) {
	buffer.WriteString(fmt.Sprintf("Session %s %.1f%% (%s remaining)\n\n",
		createProgressBar(times.ProgressPercentage, true),
		times.ProgressPercentage,
		formatTime(times.MinutesRemaining)))
}

func buildStatusBar(buffer *strings.Builder, tokens TokenMetrics, times TimeMetrics, predictedEnd time.Time, config DisplayConfig) {
	buffer.WriteString(fmt.Sprintf("Tokens: %s/%s  Estimate: %s  Reset: %s  ",
		formatNumber(tokens.Used), formatNumber(tokens.Limit),
		predictedEnd.In(config.Timezone).Format("15:04"),
		times.SessionEndTime.In(config.Timezone).Format("15:04")))

	// Status message
	if tokens.Used > tokens.Limit {
		buffer.WriteString(color.RedString("Status: LIMIT EXCEEDED"))
	} else if predictedEnd.Before(times.SessionEndTime) {
		buffer.WriteString(color.YellowString("Status: WARNING"))
	} else {
		buffer.WriteString(color.GreenString("Status: OK"))
	}
}

func addNotifications(buffer *strings.Builder, tokens TokenMetrics, tokenLimit *int) {
	if tokens.Used > 7000 && plan == "pro" && *tokenLimit > 7000 {
		buffer.WriteString(fmt.Sprintf("\n%s",
			color.HiBlackString("Note: Auto-switched to custom_max (%s tokens)",
				formatNumber(*tokenLimit))))
	}
}

func createProgressBar(percentage float64, isTime bool) string {
	filled := int(float64(ProgressBarWidth) * percentage / 100)
	bar := strings.Repeat("|", filled) + strings.Repeat(" ", ProgressBarWidth-filled)

	if isTime {
		return fmt.Sprintf("[%s]", color.BlueString(bar[:filled])+bar[filled:])
	}

	// Token bar colors
	if percentage < 60 {
		return fmt.Sprintf("[%s]", color.GreenString(bar[:filled])+bar[filled:])
	} else if percentage < 80 {
		return fmt.Sprintf("[%s]", color.YellowString(bar[:filled])+bar[filled:])
	}
	return fmt.Sprintf("[%s]", color.RedString(bar[:filled])+bar[filled:])
}

func formatTime(minutes float64) string {
	if minutes < 0 {
		minutes = 0
	}

	if minutes < 60 {
		return fmt.Sprintf("%dm", int(minutes))
	}

	hours := int(minutes / 60)
	mins := int(minutes) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

func calculateHourlyBurnRate(blocks []Block, currentTime time.Time) float64 {
	if len(blocks) == 0 {
		return 0
	}

	oneHourAgo := currentTime.Add(-time.Hour)
	totalTokens := 0.0

	for _, block := range blocks {
		if block.IsGap {
			continue
		}

		tokens := calculateBlockTokensInHour(block, currentTime, oneHourAgo)
		totalTokens += tokens
	}

	return totalTokens / 60
}

func calculateBlockTokensInHour(block Block, currentTime, oneHourAgo time.Time) float64 {
	startTime, err := time.Parse(time.RFC3339, block.StartTime)
	if err != nil {
		return 0
	}

	sessionEnd := getSessionEndTime(block, currentTime)
	if sessionEnd.Before(oneHourAgo) {
		return 0
	}

	// Calculate overlap with last hour
	sessionStartInHour := maxTime(startTime, oneHourAgo)
	sessionEndInHour := minTime(sessionEnd, currentTime)

	if sessionEndInHour.Before(sessionStartInHour) || sessionEndInHour.Equal(sessionStartInHour) {
		return 0
	}

	// Calculate portion of tokens in the last hour
	totalDuration := sessionEnd.Sub(startTime).Minutes()
	hourDuration := sessionEndInHour.Sub(sessionStartInHour).Minutes()

	if totalDuration > 0 {
		return float64(block.TotalTokens) * (hourDuration / totalDuration)
	}
	return 0
}

func getSessionEndTime(block Block, currentTime time.Time) time.Time {
	if block.IsActive {
		return currentTime
	}

	if block.ActualEndTime != "" {
		endTime, err := time.Parse(time.RFC3339, block.ActualEndTime)
		if err == nil {
			return endTime
		}
	}

	return currentTime
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// Removed getNextResetTime as it's no longer needed with rolling 5-hour sessions

func getTimezone() *time.Location {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Tokyo")
	}
	return loc
}

func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}

	// Insert commas from right to left
	result := ""
	for i, digit := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}
	return result
}
