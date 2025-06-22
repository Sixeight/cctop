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

	"github.com/spf13/cobra"
)

// Moved constants to constants.go

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

// DisplayConfig holds display configuration
type DisplayConfig struct {
	CurrentTime time.Time
	Timezone    *time.Location
	BurnRate    float64
}

// Block represents a usage block from ccusage
type Block struct {
	StartTime     string   `json:"startTime"`
	ActualEndTime string   `json:"actualEndTime"`
	TotalTokens   int      `json:"totalTokens"`
	Entries       int      `json:"entries"`
	IsActive      bool     `json:"isActive"`
	IsGap         bool     `json:"isGap"`
	Models        []string `json:"models"`
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

// SessionData represents session data from ccusage session command
type SessionData struct {
	Sessions []SessionInfo `json:"sessions"`
}

// SessionInfo represents individual session information
type SessionInfo struct {
	SessionID       string           `json:"sessionId"`
	InputTokens     int              `json:"inputTokens"`
	OutputTokens    int              `json:"outputTokens"`
	TotalTokens     int              `json:"totalTokens"`
	TotalCost       float64          `json:"totalCost"`
	LastActivity    string           `json:"lastActivity"`
	ModelsUsed      []string         `json:"modelsUsed"`
	ModelBreakdowns []ModelBreakdown `json:"modelBreakdowns"`
}

// ModelBreakdown represents per-model usage breakdown
type ModelBreakdown struct {
	ModelName    string  `json:"modelName"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	Cost         float64 `json:"cost"`
}

// Moved to session.go and display.go

var (
	config    *Config
	estimator *TokenLimitEstimator
	display   *Display
	burnCalc  *BurnRateCalculator
)

var rootCmd = &cobra.Command{
	Use:   "cctop",
	Short: "Claude Code Usage Monitor - Real-time token usage monitoring",
	Long:  `A beautiful real-time terminal monitoring tool for Claude AI token usage.`,
	Run:   runMonitor,
}

func init() {
	config = NewConfig()

	rootCmd.Flags().StringVar(&config.Plan, "plan", config.Plan, "Claude plan type (auto, pro, max5, max20)")
	rootCmd.Flags().StringVar(&config.Timezone, "timezone", config.Timezone, "Timezone for display")

	// Add analyze command for testing
	rootCmd.AddCommand(&cobra.Command{
		Use:   "analyze",
		Short: "Analyze token limit estimation accuracy",
		Run: func(cmd *cobra.Command, args []string) {
			testAccuracy()
		},
	})
}

func main() {
	estimator = NewTokenLimitEstimator()
	display = NewDisplay(config.Timezone)
	burnCalc = NewBurnRateCalculator()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Terminal control functions moved to utils.go

func runMonitor(cmd *cobra.Command, args []string) {
	hideCursor()
	defer showCursor()

	setupSignalHandler()
	tokenLimit := getInitialTokenLimit()
	clearScreen()

	for {
		if err := updateDisplay(&tokenLimit); err != nil {
			displayError(err.Error())
			time.Sleep(config.UpdateInterval)
			continue
		}
		time.Sleep(config.UpdateInterval)
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

	// Create session with all metrics
	session := NewSession(activeBlock, usageData.Blocks, *tokenLimit, time.Now())

	// Auto-switch plan if needed
	if config.ShouldAutoSwitch(config.Plan, session.Block.TotalTokens) {
		newLimit := estimator.EstimateLimit("auto", usageData.Blocks)
		if newLimit > *tokenLimit {
			*tokenLimit = newLimit
			session.Metrics.Tokens = session.calculateTokenMetrics(*tokenLimit)
		}
	}

	// Render display
	output := display.Render(session, estimator, config.Plan)
	clearAndHome()
	fmt.Print(output)
	return nil
}

func displayError(message string) {
	clearAndHome()
	fmt.Print(display.RenderError(message))
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
	data := fetchUsageData()
	if data != nil {
		return estimator.EstimateLimit(config.Plan, data.Blocks)
	}
	// Fallback to default limits if no data available
	return config.GetTokenLimit(config.Plan)
}

// Removed getTokenLimit - now using config.GetTokenLimit and estimator directly

// Removed buildDisplay - now using display.Render

// Removed calculateTokenMetrics - now in session.go

// Removed createDisplayConfig - now handled in display.go

// Removed calculateTimeMetrics - now in session.go

// Removed calculatePredictedEnd - now in session.go

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

// Removed buildHeader - now in display.go

// Removed buildTokenBar - now in display.go

// Removed buildTimeBar - now in display.go

// Removed buildStatusBar - now in display.go

// Removed addNotifications - now in display.go

// Removed createProgressBar - now in display.go

// Removed formatTime - now in utils.go

// calculateHourlyBurnRate delegates to BurnRateCalculator
func calculateHourlyBurnRate(blocks []Block, currentTime time.Time) float64 {
	if burnCalc == nil {
		burnCalc = NewBurnRateCalculator()
	}
	return burnCalc.Calculate(blocks, currentTime)
}

// Removed calculateBlockTokensInHour - now in burnrate.go

// Removed getSessionEndTime - now in burnrate.go

// Removed maxTime and minTime - now in utils.go

// Removed getTimezone - now handled in display.go

// fetchCurrentSessionData fetches session data from ccusage
func fetchCurrentSessionData() *SessionData {
	cmd := exec.Command("ccusage", "session", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var data SessionData
	if err := json.Unmarshal(output, &data); err != nil {
		return nil
	}

	return &data
}

// getCurrentWorkingDir gets the current working directory for session matching
func getCurrentWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	// Convert path separators for session ID matching
	return strings.ReplaceAll(wd, "/", "-")
}
