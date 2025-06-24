package main

import (
	"testing"
)

func TestTokenLimitEstimator(t *testing.T) {
	est := NewTokenLimitEstimator()

	tests := []struct {
		name     string
		plan     string
		blocks   []Block
		expected int
		minValue int
		maxValue int
	}{
		{
			name:     "Pro plan with no history",
			plan:     "pro",
			blocks:   nil,
			expected: 6750, // 45 * 150
			minValue: 6000,
			maxValue: 7500,
		},
		{
			name:     "Max5 plan with no history",
			plan:     "max5",
			blocks:   nil,
			expected: 33750, // 225 * 150
			minValue: 30000,
			maxValue: 37000,
		},
		{
			name: "Pro plan with history",
			plan: "pro",
			blocks: []Block{
				{TotalTokens: 5000, Entries: 40, IsGap: false, IsActive: false},
				{TotalTokens: 6500, Entries: 45, IsGap: false, IsActive: false},
				{TotalTokens: 7200, Entries: 50, IsGap: false, IsActive: false},
				{TotalTokens: 6800, Entries: 48, IsGap: false, IsActive: false},
			},
			minValue: 6000,
			maxValue: 8640, // Updated to account for max session (7200/50=144) * 45 messages
		},
		{
			name: "Max20 plan with varied token usage",
			plan: "max20",
			blocks: []Block{
				{TotalTokens: 100000, Entries: 800, IsGap: false, IsActive: false},
				{TotalTokens: 120000, Entries: 850, IsGap: false, IsActive: false},
				{TotalTokens: 135000, Entries: 900, IsGap: false, IsActive: false},
				{TotalTokens: 140000, Entries: 920, IsGap: false, IsActive: false},
				{TotalTokens: 145000, Entries: 950, IsGap: false, IsActive: false}, // outlier
			},
			minValue: 130000,
			maxValue: 145000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := est.EstimateLimit(tt.plan, tt.blocks)

			if tt.expected > 0 && result != tt.expected {
				// For exact matches
				if result < tt.minValue || result > tt.maxValue {
					t.Errorf("EstimateLimit() = %d, expected between %d and %d",
						result, tt.minValue, tt.maxValue)
				}
			} else if result < tt.minValue || result > tt.maxValue {
				// For range checks
				t.Errorf("EstimateLimit() = %d, expected between %d and %d",
					result, tt.minValue, tt.maxValue)
			}
		})
	}
}

func TestCalculateAvgTokensPerMessage(t *testing.T) {
	est := NewTokenLimitEstimator()

	tests := []struct {
		name     string
		blocks   []Block
		expected int
	}{
		{
			name: "Multiple sessions with different consumption",
			blocks: []Block{
				{TotalTokens: 5000, Entries: 40, IsGap: false, IsActive: false}, // 125 per msg
				{TotalTokens: 7200, Entries: 50, IsGap: false, IsActive: false}, // 144 per msg (highest)
				{TotalTokens: 6000, Entries: 48, IsGap: false, IsActive: false}, // 125 per msg
			},
			expected: 144, // Should use the highest consuming session
		},
		{
			name: "Single session",
			blocks: []Block{
				{TotalTokens: 8000, Entries: 50, IsGap: false, IsActive: false},
			},
			expected: 160,
		},
		{
			name: "Active session included",
			blocks: []Block{
				{TotalTokens: 5000, Entries: 40, IsGap: true, IsActive: false},
				{TotalTokens: 6000, Entries: 50, IsGap: false, IsActive: true},
			},
			expected: 120, // Now includes active sessions (6000/50)
		},
		{
			name:     "Empty blocks",
			blocks:   []Block{},
			expected: 0,
		},
		{
			name: "Session with zero entries",
			blocks: []Block{
				{TotalTokens: 5000, Entries: 0, IsGap: false, IsActive: false},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := est.calculateAvgTokensPerMessage(tt.blocks)
			if result != tt.expected {
				t.Errorf("calculateAvgTokensPerMessage() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestCalculatePercentile(t *testing.T) {
	tests := []struct {
		name       string
		values     []int
		percentile float64
		expected   int
	}{
		{
			name:       "95th percentile of varied values",
			values:     []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
			percentile: 95,
			expected:   100,
		},
		{
			name:       "50th percentile (median)",
			values:     []int{1, 2, 3, 4, 5},
			percentile: 50,
			expected:   3,
		},
		{
			name:       "Empty slice",
			values:     []int{},
			percentile: 95,
			expected:   0,
		},
		{
			name:       "Single value",
			values:     []int{42},
			percentile: 95,
			expected:   42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			est := NewTokenLimitEstimator()
			result := est.calculatePercentile(tt.values, tt.percentile)
			if result != tt.expected {
				t.Errorf("calculatePercentile() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestGetAccuracyReport(t *testing.T) {
	est := NewTokenLimitEstimator()

	tests := []struct {
		name           string
		plan           string
		actualTokens   int
		estimatedLimit int
		expectWarning  bool
	}{
		{
			name:           "Accurate estimation",
			plan:           "pro",
			actualTokens:   6800,
			estimatedLimit: 7000,
			expectWarning:  false,
		},
		{
			name:           "Inaccurate estimation - over 10%",
			plan:           "pro",
			actualTokens:   8000,
			estimatedLimit: 7000,
			expectWarning:  true,
		},
		{
			name:           "Inaccurate estimation - under 10%",
			plan:           "max5",
			actualTokens:   30000,
			estimatedLimit: 35000,
			expectWarning:  true,
		},
		{
			name:           "Zero estimated limit",
			plan:           "pro",
			actualTokens:   5000,
			estimatedLimit: 0,
			expectWarning:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := est.GetAccuracyReport(tt.plan, tt.actualTokens, tt.estimatedLimit)
			hasWarning := report != ""

			if hasWarning != tt.expectWarning {
				t.Errorf("GetAccuracyReport() warning = %v, expected %v\nReport: %s",
					hasWarning, tt.expectWarning, report)
			}
		})
	}
}
