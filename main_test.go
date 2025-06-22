package main

import (
	"testing"
	"time"
)

func TestGetTokenLimit(t *testing.T) {
	tests := []struct {
		name     string
		planType string
		blocks   []Block
		expected int
	}{
		{
			name:     "Pro plan",
			planType: "pro",
			blocks:   nil,
			expected: 7000,
		},
		{
			name:     "Max5 plan",
			planType: "max5",
			blocks:   nil,
			expected: 35000,
		},
		{
			name:     "Max20 plan",
			planType: "max20",
			blocks:   nil,
			expected: 140000,
		},
		{
			name:     "Custom max with blocks",
			planType: "custom_max",
			blocks: []Block{
				{TotalTokens: 5000, IsGap: false, IsActive: false},
				{TotalTokens: 8000, IsGap: false, IsActive: false},
				{TotalTokens: 3000, IsGap: false, IsActive: true}, // Active, should be ignored
			},
			expected: 8000,
		},
		{
			name:     "Custom max without blocks",
			planType: "custom_max",
			blocks:   nil,
			expected: 7000, // Default to pro
		},
		{
			name:     "Invalid plan",
			planType: "invalid",
			blocks:   nil,
			expected: 7000, // Default to pro
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTokenLimit(tt.planType, tt.blocks)
			if result != tt.expected {
				t.Errorf("getTokenLimit(%s) = %d, expected %d", tt.planType, result, tt.expected)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		minutes  float64
		expected string
	}{
		{
			name:     "Less than hour",
			minutes:  45,
			expected: "45m",
		},
		{
			name:     "Exactly one hour",
			minutes:  60,
			expected: "1h",
		},
		{
			name:     "Hour and minutes",
			minutes:  125,
			expected: "2h5m",
		},
		{
			name:     "Multiple hours exact",
			minutes:  180,
			expected: "3h",
		},
		{
			name:     "Negative time",
			minutes:  -30,
			expected: "0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.minutes)
			if result != tt.expected {
				t.Errorf("formatTime(%.1f) = %s, expected %s", tt.minutes, result, tt.expected)
			}
		})
	}
}

func TestCalculateHourlyBurnRate(t *testing.T) {
	currentTime := time.Now()

	tests := []struct {
		name     string
		blocks   []Block
		expected float64
	}{
		{
			name:     "No blocks",
			blocks:   []Block{},
			expected: 0,
		},
		{
			name: "Active session in last hour",
			blocks: []Block{
				{
					StartTime:   currentTime.Add(-30 * time.Minute).Format(time.RFC3339),
					TotalTokens: 600,
					IsActive:    true,
					IsGap:       false,
				},
			},
			expected: 10.0, // 600 tokens / 60 minutes
		},
		{
			name: "Multiple sessions",
			blocks: []Block{
				{
					StartTime:     currentTime.Add(-45 * time.Minute).Format(time.RFC3339),
					ActualEndTime: currentTime.Add(-15 * time.Minute).Format(time.RFC3339),
					TotalTokens:   300,
					IsActive:      false,
					IsGap:         false,
				},
				{
					StartTime:   currentTime.Add(-10 * time.Minute).Format(time.RFC3339),
					TotalTokens: 100,
					IsActive:    true,
					IsGap:       false,
				},
			},
			expected: 6.67, // More accurate calculation
		},
		{
			name: "Gap blocks should be ignored",
			blocks: []Block{
				{
					StartTime:   currentTime.Add(-30 * time.Minute).Format(time.RFC3339),
					TotalTokens: 1000,
					IsActive:    false,
					IsGap:       true, // Should be ignored
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateHourlyBurnRate(tt.blocks, currentTime)
			// Allow for some floating point variance
			if result < tt.expected-1.0 || result > tt.expected+1.0 {
				t.Errorf("calculateHourlyBurnRate() = %.2f, expected %.2f", result, tt.expected)
			}
		})
	}
}


func TestCreateProgressBars(t *testing.T) {
	// Test progress bar for tokens
	bar := createProgressBar(50.0, false)
	if len(bar) == 0 {
		t.Error("createProgressBar returned empty string for token bar")
	}

	// Test progress bar for time
	bar = createProgressBar(50.0, true)
	if len(bar) == 0 {
		t.Error("createProgressBar returned empty string for time bar")
	}
}
