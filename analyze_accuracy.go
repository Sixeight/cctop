package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

// AccuracyAnalysis analyzes the accuracy of token limit estimation
type AccuracyAnalysis struct {
	Plan                string
	ActualMax           int
	EstimatedLimit      int
	AccuracyPercentage  float64
	SampleSize          int
	AverageTokensPerMsg int
	StdDeviation        float64
}

func analyzeEstimationAccuracy() {
	// Fetch usage data
	cmd := exec.Command("ccusage", "blocks", "--json")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error fetching usage data:", err)
		return
	}

	var data CCUsageData
	if err := json.Unmarshal(output, &data); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Initialize estimator
	estimator := NewTokenLimitEstimator()

	// Analyze for each plan
	plans := []string{"pro", "max5", "max20"}
	fmt.Println("Token Limit Estimation Accuracy Analysis")
	fmt.Println("========================================")

	for _, plan := range plans {
		analysis := performAnalysis(plan, data.Blocks, estimator)
		printAnalysis(analysis)
	}

	// Analyze token per message variance
	analyzeTokenPerMessageVariance(data.Blocks)
}

func performAnalysis(plan string, blocks []Block, estimator *TokenLimitEstimator) AccuracyAnalysis {
	// Get estimated limit
	estimated := estimator.EstimateLimit(plan, blocks)

	// Calculate actual max from historical data
	var sessionTokens []int
	var totalTokens, totalEntries int

	for _, block := range blocks {
		if !block.IsGap && !block.IsActive && block.TotalTokens > 0 {
			sessionTokens = append(sessionTokens, block.TotalTokens)
			totalTokens += block.TotalTokens
			totalEntries += block.Entries
		}
	}

	actualMax := 0
	if len(sessionTokens) > 0 {
		actualMax = calculatePercentile(sessionTokens, 95)
	}

	avgTokensPerMsg := 0
	if totalEntries > 0 {
		avgTokensPerMsg = totalTokens / totalEntries
	}

	// Calculate standard deviation
	stdDev := calculateStdDev(sessionTokens)

	// Calculate accuracy
	accuracy := 100.0
	if actualMax > 0 {
		diff := math.Abs(float64(estimated-actualMax)) / float64(actualMax) * 100
		accuracy = 100.0 - diff
	}

	return AccuracyAnalysis{
		Plan:                plan,
		ActualMax:           actualMax,
		EstimatedLimit:      estimated,
		AccuracyPercentage:  accuracy,
		SampleSize:          len(sessionTokens),
		AverageTokensPerMsg: avgTokensPerMsg,
		StdDeviation:        stdDev,
	}
}

func calculateStdDev(values []int) float64 {
	if len(values) < 2 {
		return 0
	}

	// Calculate mean
	sum := 0
	for _, v := range values {
		sum += v
	}
	mean := float64(sum) / float64(len(values))

	// Calculate variance
	variance := 0.0
	for _, v := range values {
		diff := float64(v) - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

func printAnalysis(a AccuracyAnalysis) {
	fmt.Printf("Plan: %s\n", a.Plan)
	fmt.Printf("├─ Sample Size: %d sessions\n", a.SampleSize)
	fmt.Printf("├─ Actual 95th Percentile: %s tokens\n", formatNumber(a.ActualMax))
	fmt.Printf("├─ Estimated Limit: %s tokens\n", formatNumber(a.EstimatedLimit))
	fmt.Printf("├─ Accuracy: %.1f%%\n", a.AccuracyPercentage)
	fmt.Printf("├─ Avg Tokens/Message: %d\n", a.AverageTokensPerMsg)
	fmt.Printf("└─ Std Deviation: %.0f tokens\n\n", a.StdDeviation)
}

func analyzeTokenPerMessageVariance(blocks []Block) {
	fmt.Println("Token Per Message Variance Analysis")
	fmt.Println("===================================")

	var ratios []float64
	for _, block := range blocks {
		if !block.IsGap && !block.IsActive && block.Entries > 0 {
			ratio := float64(block.TotalTokens) / float64(block.Entries)
			ratios = append(ratios, ratio)
		}
	}

	if len(ratios) == 0 {
		fmt.Println("No data available for analysis")
		return
	}

	// Calculate statistics
	minVal, maxVal := ratios[0], ratios[0]
	sum := 0.0
	for _, r := range ratios {
		if r < minVal {
			minVal = r
		}
		if r > maxVal {
			maxVal = r
		}
		sum += r
	}
	avg := sum / float64(len(ratios))

	fmt.Printf("Token/Message Statistics:\n")
	fmt.Printf("├─ Minimum: %.1f tokens/msg\n", minVal)
	fmt.Printf("├─ Maximum: %.1f tokens/msg\n", maxVal)
	fmt.Printf("├─ Average: %.1f tokens/msg\n", avg)
	fmt.Printf("└─ Variance Range: %.1fx\n\n", maxVal/minVal)

	// Explain implications
	fmt.Println("Implications:")
	if maxVal/minVal > 3 {
		fmt.Println("⚠️  High variance (>3x) in token usage per message")
		fmt.Println("   This makes static estimation less reliable")
		fmt.Println("   Dynamic learning approach is recommended")
	} else {
		fmt.Println("✓  Moderate variance in token usage")
		fmt.Println("   Both static and dynamic approaches should work well")
	}
}

// Add this to main.go temporarily for testing
func testAccuracy() {
	analyzeEstimationAccuracy()
}
