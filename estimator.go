package main

import (
	"fmt"
	"math"
	"sort"
)

// TokenLimitEstimator manages dynamic token limit estimation
type TokenLimitEstimator struct {
	// Base limits from official documentation (messages * avg tokens)
	baseLimits map[string]BaseLimit
}

// BaseLimit represents official plan limits
type BaseLimit struct {
	Messages            int
	DefaultTokensPerMsg int
}

// NewTokenLimitEstimator creates a new estimator with official limits
func NewTokenLimitEstimator() *TokenLimitEstimator {
	return &TokenLimitEstimator{
		baseLimits: map[string]BaseLimit{
			"pro":   {Messages: 45, DefaultTokensPerMsg: 150},
			"max5":  {Messages: 225, DefaultTokensPerMsg: 150},
			"max20": {Messages: 900, DefaultTokensPerMsg: 150},
		},
	}
}

// EstimateLimit estimates token limit using historical data and official limits
func (e *TokenLimitEstimator) EstimateLimit(plan string, blocks []Block) int {
	// First try dynamic estimation from historical data
	if dynamicLimit := e.estimateFromHistory(blocks); dynamicLimit > 0 {
		// If we have historical data, use hybrid approach
		if baseLimit := e.calculateBaseLimit(plan, blocks); baseLimit > 0 {
			// Adaptive weighting based on sample size and variance
			weight := e.calculateDynamicWeight(blocks)
			return int(float64(dynamicLimit)*weight + float64(baseLimit)*(1-weight))
		}
		return dynamicLimit
	}

	// Fallback to base calculation
	return e.calculateBaseLimit(plan, blocks)
}

// estimateFromHistory analyzes historical session data
func (e *TokenLimitEstimator) estimateFromHistory(blocks []Block) int {
	var sessionMaxTokens []int

	for _, block := range blocks {
		if !block.IsGap && !block.IsActive && block.TotalTokens > 0 {
			sessionMaxTokens = append(sessionMaxTokens, block.TotalTokens)
		}
	}

	if len(sessionMaxTokens) < 5 {
		// Not enough data for reliable estimation
		return 0
	}

	// Remove extreme outliers using IQR method
	cleaned := removeOutliers(sessionMaxTokens)
	if len(cleaned) < 3 {
		// If too many outliers removed, use 85th percentile of original
		return calculatePercentile(sessionMaxTokens, 85)
	}

	// Use 90th percentile of cleaned data (more conservative than 95th)
	return calculatePercentile(cleaned, 90)
}

// removeOutliers removes values outside 1.5 * IQR
func removeOutliers(values []int) []int {
	if len(values) < 4 {
		return values
	}

	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

	q1 := calculatePercentile(sorted, 25)
	q3 := calculatePercentile(sorted, 75)
	iqr := q3 - q1

	lowerBound := q1 - int(1.5*float64(iqr))
	upperBound := q3 + int(1.5*float64(iqr))

	var cleaned []int
	for _, v := range values {
		if v >= lowerBound && v <= upperBound {
			cleaned = append(cleaned, v)
		}
	}

	return cleaned
}

// calculateBaseLimit calculates limit based on official message counts
func (e *TokenLimitEstimator) calculateBaseLimit(plan string, blocks []Block) int {
	base, exists := e.baseLimits[plan]
	if !exists {
		// Default to pro plan
		base = e.baseLimits["pro"]
	}

	// Calculate actual tokens per message from recent data
	avgTokensPerMsg := e.calculateAvgTokensPerMessage(blocks)
	if avgTokensPerMsg > 0 {
		return base.Messages * avgTokensPerMsg
	}

	// Use default tokens per message
	return base.Messages * base.DefaultTokensPerMsg
}

// calculateAvgTokensPerMessage calculates average tokens per message from recent sessions
func (e *TokenLimitEstimator) calculateAvgTokensPerMessage(blocks []Block) int {
	var totalTokens, totalEntries int

	// Only use recent complete sessions (last 10)
	count := 0
	for i := len(blocks) - 1; i >= 0 && count < 10; i-- {
		block := blocks[i]
		if !block.IsGap && !block.IsActive && block.Entries > 0 {
			totalTokens += block.TotalTokens
			totalEntries += block.Entries
			count++
		}
	}

	if totalEntries == 0 {
		return 0
	}

	return totalTokens / totalEntries
}

// calculatePercentile calculates the nth percentile of a slice of integers
func calculatePercentile(values []int, percentile float64) int {
	if len(values) == 0 {
		return 0
	}

	sort.Ints(values)
	index := int(math.Ceil(float64(len(values))*percentile/100.0)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}

	return values[index]
}

// GetAccuracyReport generates a report on estimation accuracy
func (e *TokenLimitEstimator) GetAccuracyReport(plan string, actualTokens, estimatedLimit int) string {
	if estimatedLimit == 0 {
		return ""
	}

	deviation := float64(actualTokens-estimatedLimit) / float64(estimatedLimit) * 100

	if math.Abs(deviation) > 10 {
		return fmt.Sprintf("Warning: Token limit estimation may be inaccurate (deviation: %.1f%%)", deviation)
	}

	return ""
}

// calculateDynamicWeight determines how much to trust historical data
func (e *TokenLimitEstimator) calculateDynamicWeight(blocks []Block) float64 {
	var sessionTokens []int
	for _, block := range blocks {
		if !block.IsGap && !block.IsActive && block.TotalTokens > 0 {
			sessionTokens = append(sessionTokens, block.TotalTokens)
		}
	}

	sampleSize := len(sessionTokens)

	// Less weight for small sample sizes
	if sampleSize < 10 {
		return 0.3
	} else if sampleSize < 20 {
		return 0.5
	}

	// Calculate coefficient of variation (CV)
	if len(sessionTokens) > 1 {
		mean := 0
		for _, v := range sessionTokens {
			mean += v
		}
		mean /= len(sessionTokens)

		variance := 0.0
		for _, v := range sessionTokens {
			diff := float64(v - mean)
			variance += diff * diff
		}
		variance /= float64(len(sessionTokens))
		stdDev := math.Sqrt(variance)

		cv := stdDev / float64(mean)

		// High variance = less trust in historical data
		if cv > 0.5 {
			return 0.4
		} else if cv > 0.3 {
			return 0.6
		}
	}

	return 0.8 // High confidence with good data
}
