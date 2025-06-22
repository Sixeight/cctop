package main

import (
	"time"
)

// Config holds all application configuration
type Config struct {
	TokenLimits    map[string]int
	Plan           string
	Timezone       string
	Thresholds     ThresholdConfig
	ProgressBar    ProgressBarConfig
	UpdateInterval time.Duration
}

// ProgressBarConfig holds progress bar configuration
type ProgressBarConfig struct {
	Width            int
	TokenColorLow    float64 // Percentage threshold for green
	TokenColorMedium float64 // Percentage threshold for yellow
}

// ThresholdConfig holds various threshold values
type ThresholdConfig struct {
	MinHistoricalSessions  int     // Minimum sessions for reliable estimation
	OutlierIQRMultiplier   float64 // IQR multiplier for outlier detection
	HistoricalPercentile   float64 // Percentile for historical estimation
	AccuracyWarningPercent float64 // Deviation percentage for accuracy warning
	AutoSwitchTokens       int     // Token threshold for auto plan switching
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		Plan:           "auto",
		Timezone:       "Asia/Tokyo",
		UpdateInterval: 3 * time.Second,
		TokenLimits: map[string]int{
			"pro":   7000,
			"max5":  35000,
			"max20": 140000,
		},
		ProgressBar: ProgressBarConfig{
			Width:            50,
			TokenColorLow:    60,
			TokenColorMedium: 80,
		},
		Thresholds: ThresholdConfig{
			MinHistoricalSessions:  5,
			OutlierIQRMultiplier:   1.5,
			HistoricalPercentile:   90,
			AccuracyWarningPercent: 10,
			AutoSwitchTokens:       7000,
		},
	}
}

// GetTokenLimit returns the token limit for a given plan
func (c *Config) GetTokenLimit(plan string) int {
	if limit, ok := c.TokenLimits[plan]; ok {
		return limit
	}
	return c.TokenLimits["pro"] // Default to pro plan
}

// ShouldAutoSwitch checks if auto-switching should occur
func (c *Config) ShouldAutoSwitch(currentPlan string, tokensUsed int) bool {
	return currentPlan == "pro" && tokensUsed > c.Thresholds.AutoSwitchTokens
}

// GetProgressBarColor returns the color name based on percentage
func (c *Config) GetProgressBarColor(percentage float64) string {
	if percentage < c.ProgressBar.TokenColorLow {
		return "green"
	} else if percentage < c.ProgressBar.TokenColorMedium {
		return "yellow"
	}
	return "red"
}

// ValidatePlan ensures the plan is valid
func (c *Config) ValidatePlan() {
	validPlans := map[string]bool{
		"auto":  true,
		"pro":   true,
		"max5":  true,
		"max20": true,
	}

	if !validPlans[c.Plan] {
		c.Plan = "auto"
	}
}
