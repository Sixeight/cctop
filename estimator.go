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
			"pro":   {Messages: ProPlanMessages, DefaultTokensPerMsg: DefaultTokensPerMsg},
			"max5":  {Messages: Max5PlanMessages, DefaultTokensPerMsg: DefaultTokensPerMsg},
			"max20": {Messages: Max20PlanMessages, DefaultTokensPerMsg: DefaultTokensPerMsg},
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

	if len(sessionMaxTokens) < MinHistoricalSessions {
		// Not enough data for reliable estimation
		return 0
	}

	// Remove extreme outliers using IQR method
	cleaned := e.removeOutliers(sessionMaxTokens)
	if len(cleaned) < MinCleanedSessions {
		// If too many outliers removed, use fallback percentile of original
		return e.calculatePercentile(sessionMaxTokens, FallbackPercentile)
	}

	// Use historical percentile of cleaned data
	return e.calculatePercentile(cleaned, HistoricalPercentile)
}

// removeOutliers removes values outside 1.5 * IQR
func (e *TokenLimitEstimator) removeOutliers(values []int) []int {
	if len(values) < 4 {
		return values
	}

	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

	q1 := e.calculatePercentile(sorted, 25)
	q3 := e.calculatePercentile(sorted, 75)
	iqr := q3 - q1

	lowerBound := q1 - int(OutlierIQRMultiplier*float64(iqr))
	upperBound := q3 + int(OutlierIQRMultiplier*float64(iqr))

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
	// For auto plan, detect appropriate plan level from historical data
	if plan == "auto" {
		plan = e.detectPlanFromHistory(blocks)
	}

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

// detectPlanFromHistory detects the appropriate plan based on historical usage
func (e *TokenLimitEstimator) detectPlanFromHistory(blocks []Block) string {
	var maxTokens int
	for _, block := range blocks {
		if !block.IsGap && !block.IsActive && block.TotalTokens > maxTokens {
			maxTokens = block.TotalTokens
		}
	}

	// Detect plan based on historical max usage
	switch {
	case maxTokens > Max20DetectionThreshold:
		return "max20"
	case maxTokens > Max5DetectionThreshold:
		return "max5"
	default:
		return "pro"
	}
}

// calculateAvgTokensPerMessage calculates average tokens per message from recent sessions
func (e *TokenLimitEstimator) calculateAvgTokensPerMessage(blocks []Block) int {
	var totalTokens, totalEntries int

	// Only use recent complete sessions
	count := 0
	for i := len(blocks) - 1; i >= 0 && count < RecentSessionsCount; i-- {
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
func (e *TokenLimitEstimator) calculatePercentile(values []int, percentile float64) int {
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

// calculateDeviation calculates the percentage deviation between actual and estimated values
func (e *TokenLimitEstimator) calculateDeviation(actual, estimated int) float64 {
	if estimated == 0 {
		return 0
	}
	return float64(actual-estimated) / float64(estimated) * 100
}

// formatAccuracyWarning generates a warning message if deviation exceeds threshold
func (e *TokenLimitEstimator) formatAccuracyWarning(deviation float64, isAverage bool) string {
	if math.Abs(deviation) > AccuracyWarningThreshold {
		if isAverage {
			return fmt.Sprintf("Warning: Token limit estimation may be inaccurate (avg deviation: %.1f%%)", deviation)
		}
		return fmt.Sprintf("Warning: Token limit estimation may be inaccurate (deviation: %.1f%%)", deviation)
	}
	return ""
}

// GetAccuracyReport generates a report on estimation accuracy
func (e *TokenLimitEstimator) GetAccuracyReport(plan string, actualTokens, estimatedLimit int) string {
	if estimatedLimit == 0 {
		return ""
	}

	deviation := e.calculateDeviation(actualTokens, estimatedLimit)
	return e.formatAccuracyWarning(deviation, false)
}

// GetHistoricalAccuracyReport evaluates estimation accuracy based on historical data
func (e *TokenLimitEstimator) GetHistoricalAccuracyReport(plan string, blocks []Block, currentEstimatedLimit int) string {
	if currentEstimatedLimit == 0 || len(blocks) < MinHistoricalSessions {
		return ""
	}

	// Collect completed sessions for accuracy analysis
	var deviations []float64
	for _, block := range blocks {
		if !block.IsGap && !block.IsActive && block.TotalTokens > 0 {
			// Calculate what the limit would have been estimated for this historical session
			historicalEstimate := e.EstimateLimit(plan, blocks)
			if historicalEstimate > 0 {
				deviation := e.calculateDeviation(block.TotalTokens, historicalEstimate)
				deviations = append(deviations, math.Abs(deviation))
			}
		}
	}

	if len(deviations) < MinHistoricalSessions {
		return ""
	}

	// Calculate average deviation
	avgDeviation := 0.0
	for _, d := range deviations {
		avgDeviation += d
	}
	avgDeviation /= float64(len(deviations))

	return e.formatAccuracyWarning(avgDeviation, true)
}

// GetActualPlan returns the actual plan being used (resolves 'auto' to detected plan)
func (e *TokenLimitEstimator) GetActualPlan(plan string, blocks []Block) string {
	if plan == "auto" {
		return e.detectPlanFromHistory(blocks)
	}
	return plan
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
		return WeightSmallSample
	} else if sampleSize < 20 {
		return WeightMediumSample
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
		if cv > VarianceCoefficientHigh {
			return WeightHighVariance
		} else if cv > VarianceCoefficientMedium {
			return WeightMediumVariance
		}
	}

	return WeightLargeSample // High confidence with good data
}
