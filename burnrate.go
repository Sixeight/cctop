package main

import (
	"time"
)

// BurnRateCalculator calculates token burn rate over a time window
type BurnRateCalculator struct {
	window time.Duration
}

// NewBurnRateCalculator creates a new calculator with a 1-hour window
func NewBurnRateCalculator() *BurnRateCalculator {
	return &BurnRateCalculator{
		window: time.Hour,
	}
}

// Calculate computes the burn rate in tokens per minute
func (b *BurnRateCalculator) Calculate(blocks []Block, currentTime time.Time) float64 {
	if len(blocks) == 0 {
		return 0
	}

	windowStart := currentTime.Add(-b.window)
	totalTokens := 0.0

	for _, block := range blocks {
		if block.IsGap {
			continue
		}

		tokens := b.calculateBlockTokensInWindow(block, currentTime, windowStart)
		totalTokens += tokens
	}

	// Convert to tokens per minute
	return totalTokens / b.window.Minutes()
}

// calculateBlockTokensInWindow calculates tokens from a block within the time window
func (b *BurnRateCalculator) calculateBlockTokensInWindow(block Block, windowEnd, windowStart time.Time) float64 {
	blockStart, err := time.Parse(time.RFC3339, block.StartTime)
	if err != nil {
		return 0
	}

	blockEnd := b.getBlockEndTime(block, windowEnd)

	// Check if block is outside the window
	if blockEnd.Before(windowStart) {
		return 0
	}

	// Calculate overlap with window
	overlapStart := maxTime(blockStart, windowStart)
	overlapEnd := minTime(blockEnd, windowEnd)

	if overlapEnd.Before(overlapStart) || overlapEnd.Equal(overlapStart) {
		return 0
	}

	// Calculate portion of tokens in the window
	totalDuration := blockEnd.Sub(blockStart).Minutes()
	overlapDuration := overlapEnd.Sub(overlapStart).Minutes()

	if totalDuration > 0 {
		return float64(block.TotalTokens) * (overlapDuration / totalDuration)
	}

	return 0
}

// getBlockEndTime determines the end time of a block
func (b *BurnRateCalculator) getBlockEndTime(block Block, currentTime time.Time) time.Time {
	if block.IsActive {
		return currentTime
	}

	if block.ActualEndTime != "" {
		endTime, err := time.Parse(time.RFC3339, block.ActualEndTime)
		if err == nil {
			return endTime
		}
	}

	// Fallback to current time if no end time is available
	return currentTime
}
