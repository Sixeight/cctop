package main

import (
	"fmt"
	"time"
)

// maxTokenSessionResult holds the result of finding max token session
type maxTokenSessionResult struct {
	block *Block
	index int
}

// findMaxTokenSession finds the session with highest token consumption
func (e *TokenLimitEstimator) findMaxTokenSession(blocks []Block) maxTokenSessionResult {
	var result maxTokenSessionResult
	maxTokens := 0
	currentIndex := 0

	for i := range blocks {
		block := &blocks[i]
		if !block.IsGap {
			currentIndex++
			if block.Entries > 0 && block.TotalTokens > maxTokens {
				maxTokens = block.TotalTokens
				result.block = block
				result.index = currentIndex
			}
		}
	}

	return result
}

// getMessageTokens retrieves message tokens from JSONL files
func (e *TokenLimitEstimator) getMessageTokens(block *Block) ([]int, error) {
	reader := NewMessageTokenReader()
	endTime := block.ActualEndTime
	if endTime == "" {
		// For active sessions, use current time
		endTime = time.Now().Format(time.RFC3339)
	}
	
	return reader.GetBlockTokens(block.StartTime, endTime)
}

// calculateTokensPerMessage calculates tokens per message using the selected method
func (e *TokenLimitEstimator) calculateTokensPerMessage(messageTokens []int, block *Block) (tokensPerMsg int, methodDesc string) {
	switch e.estimationMethod {
	case "median":
		return e.calculatePercentile(messageTokens, 50), "median"
	case "mode":
		return CalculateMode(messageTokens), "mode"
	case "avg":
		return block.TotalTokens / block.Entries, "average"
	default:
		return e.parseCustomMethod(messageTokens, block)
	}
}

// parseCustomMethod handles custom percentile and trim methods
func (e *TokenLimitEstimator) parseCustomMethod(messageTokens []int, block *Block) (tokensPerMsg int, methodDesc string) {
	// Parse percentile (e.g., "p35")
	if len(e.estimationMethod) > 1 && e.estimationMethod[0] == 'p' {
		var percentile float64
		if _, err := fmt.Sscanf(e.estimationMethod, "p%f", &percentile); err == nil && percentile >= 0 && percentile <= 100 {
			tokensPerMsg := e.calculatePercentile(messageTokens, percentile)
			if percentile == 50 {
				return tokensPerMsg, "median"
			}
			return tokensPerMsg, fmt.Sprintf("%.0fth percentile", percentile)
		}
	}
	
	// Parse trim percentage (e.g., "trim15")
	if len(e.estimationMethod) > 4 && e.estimationMethod[:4] == "trim" {
		var trimPercent float64
		if _, err := fmt.Sscanf(e.estimationMethod, "trim%f", &trimPercent); err == nil && trimPercent >= 0 && trimPercent < 50 {
			return CalculateTrimmedMean(messageTokens, trimPercent), fmt.Sprintf("%.0f%% trimmed mean", trimPercent)
		}
	}
	
	// Fallback to 40th percentile
	return e.calculatePercentile(messageTokens, 40), "40th percentile"
}