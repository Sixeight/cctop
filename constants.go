package main

import "time"

// Time-related constants
const (
	SessionDurationMinutes = 300.0           // 5 hours in minutes
	SessionDuration        = 5 * time.Hour   // 5 hours
	UpdateInterval         = 3 * time.Second // Display refresh interval
	BurnRateWindow         = 1 * time.Hour   // Window for burn rate calculation
	MinutesPerHour         = 60.0            // Minutes in an hour
)

// Display constants
const (
	ProgressBarWidth = 50           // Width of progress bars in characters
	TimeFormat       = "15:04:05"   // HH:MM:SS format
	TimeFormatShort  = "15:04"      // HH:MM format
	DateFormat       = "2006-01-02" // YYYY-MM-DD format
)

// Token limit constants
const (
	DefaultTokenLimit   = 7000 // Default token limit for unknown plans
	ProPlanMessages     = 45   // Messages allowed in Pro plan
	Max5PlanMessages    = 225  // Messages allowed in Max5 plan
	Max20PlanMessages   = 900  // Messages allowed in Max20 plan
	DefaultTokensPerMsg = 150  // Default tokens per message estimate
)

// Threshold constants
const (
	TokenColorThresholdLow    = 60.0 // Below this percentage shows green
	TokenColorThresholdMedium = 80.0 // Below this percentage shows yellow
	MinHistoricalSessions     = 5    // Minimum sessions for historical estimation
	MinCleanedSessions        = 3    // Minimum sessions after outlier removal
	OutlierIQRMultiplier      = 1.5  // IQR multiplier for outlier detection
	HistoricalPercentile      = 90.0 // Percentile for historical estimation
	FallbackPercentile        = 85.0 // Percentile when too many outliers removed
	AccuracyWarningThreshold  = 10.0 // Percentage deviation for accuracy warning
	AutoSwitchThreshold       = 7000 // Token threshold for auto plan switching
)

// Estimation weight constants
const (
	WeightSmallSample    = 0.3 // Weight for <10 historical sessions
	WeightMediumSample   = 0.5 // Weight for 10-20 historical sessions
	WeightLargeSample    = 0.8 // Weight for 20+ historical sessions
	WeightHighVariance   = 0.4 // Weight when CV > 0.5
	WeightMediumVariance = 0.6 // Weight when CV > 0.3
)

// Statistical constants
const (
	VarianceCoefficientHigh   = 0.5 // High coefficient of variation
	VarianceCoefficientMedium = 0.3 // Medium coefficient of variation
	RecentSessionsCount       = 10  // Number of recent sessions to analyze
)

// Terminal control sequences
const (
	HideCursor   = "\033[?25l"     // ANSI escape to hide cursor
	ShowCursor   = "\033[?25h"     // ANSI escape to show cursor
	ClearScreen  = "\033[2J\033[H" // Clear entire screen and move to home
	ClearAndHome = "\033[H\033[J"  // Move to home and clear to end
)

// Plan detection thresholds
const (
	Max20DetectionThreshold = 100000 // Tokens indicating Max20 plan
	Max5DetectionThreshold  = 25000  // Tokens indicating Max5 plan
)
