package main

import (
	"strings"
	"time"
)

// Session represents an active Claude session with all related data
type Session struct {
	StartTime     time.Time
	EndTime       time.Time
	Block         *Block
	AllBlocks     []Block
	PrimaryModel  string
	CurrentModels []string
	Metrics       SessionMetrics
	BurnRate      float64
	TodayCost     float64
}

// SessionMetrics contains all calculated metrics for a session
type SessionMetrics struct {
	Time   TimeMetrics
	Tokens TokenMetrics
}

// NewSession creates a new Session from an active block
func NewSession(block *Block, allBlocks []Block, tokenLimit int, currentTime time.Time) *Session {
	startTime, _ := time.Parse(time.RFC3339, block.StartTime)
	endTime := startTime.Add(5 * time.Hour)

	session := &Session{
		Block:         block,
		AllBlocks:     allBlocks,
		StartTime:     startTime,
		EndTime:       endTime,
		BurnRate:      burnCalc.Calculate(allBlocks, currentTime),
		TodayCost:     fetchTodayTotalCost(currentTime),
		CurrentModels: block.Models,
		PrimaryModel:  determinePrimaryModel(block.Models),
	}

	// Calculate metrics
	session.Metrics.Tokens = session.calculateTokenMetrics(tokenLimit)
	session.Metrics.Time = session.calculateTimeMetrics(currentTime)

	return session
}

// calculateTokenMetrics calculates token usage metrics for the session
func (s *Session) calculateTokenMetrics(limit int) TokenMetrics {
	used := s.Block.TotalTokens
	percentage := 0.0
	if limit > 0 {
		percentage = float64(used) / float64(limit) * 100
	}

	return TokenMetrics{
		Used:       used,
		Limit:      limit,
		Percentage: percentage,
		Remaining:  limit - used,
	}
}

// calculateTimeMetrics calculates time-based metrics for the session
func (s *Session) calculateTimeMetrics(currentTime time.Time) TimeMetrics {
	elapsedMinutes := currentTime.Sub(s.StartTime).Minutes()
	remainingMinutes := s.EndTime.Sub(currentTime).Minutes()
	if remainingMinutes < 0 {
		remainingMinutes = 0
	}

	progressPercentage := (elapsedMinutes / SessionDurationMinutes) * 100
	if progressPercentage < 0 {
		progressPercentage = 0
	} else if progressPercentage > 100 {
		progressPercentage = 100
	}

	return TimeMetrics{
		SessionEndTime:     s.EndTime,
		MinutesRemaining:   remainingMinutes,
		ProgressPercentage: progressPercentage,
	}
}

// GetPredictedEndTime calculates when tokens will be depleted
func (s *Session) GetPredictedEndTime(currentTime time.Time) time.Time {
	if s.BurnRate > 0 && s.Metrics.Tokens.Remaining > 0 {
		minutesToDepletion := float64(s.Metrics.Tokens.Remaining) / s.BurnRate
		return currentTime.Add(time.Duration(minutesToDepletion) * time.Minute)
	}
	return s.EndTime
}

// GetStatus returns the current status of the session
func (s *Session) GetStatus() string {
	if s.Metrics.Tokens.Used > s.Metrics.Tokens.Limit {
		return "LIMIT EXCEEDED"
	}

	predictedEnd := s.GetPredictedEndTime(time.Now())
	if predictedEnd.Before(s.EndTime) {
		return "WARNING"
	}

	return "OK"
}

// IsOverLimit returns true if token usage exceeds the limit
func (s *Session) IsOverLimit() bool {
	return s.Metrics.Tokens.Used > s.Metrics.Tokens.Limit
}

// GetStatusColor returns the appropriate color for the current status
func (s *Session) GetStatusColor() string {
	switch s.GetStatus() {
	case "LIMIT EXCEEDED":
		return "red"
	case "WARNING":
		return "yellow"
	default:
		return "green"
	}
}

// determinePrimaryModel determines the currently active model from session data
func determinePrimaryModel(models []string) string {
	if len(models) == 0 {
		return "unknown"
	}

	// Get the current session model breakdown to determine the most recently used model
	currentModel := getCurrentActiveModel()
	if currentModel != "" {
		return currentModel
	}

	// Fallback: If only one non-synthetic model, use it
	var realModels []string
	for _, model := range models {
		if model != "<synthetic>" {
			realModels = append(realModels, model)
		}
	}

	if len(realModels) == 1 {
		return formatModelName(realModels[0])
	}

	// For multiple models, assume the most recent/likely model
	// In practice, if there's usage switching, Sonnet is more likely to be current
	// because Opus typically switches TO Sonnet when limits are reached
	for _, model := range models {
		if strings.Contains(strings.ToLower(model), "sonnet") {
			return model
		}
	}

	for _, model := range models {
		if strings.Contains(strings.ToLower(model), "opus") {
			return model
		}
	}

	// Return first non-synthetic model
	for _, model := range models {
		if model != "<synthetic>" {
			return formatModelName(model)
		}
	}

	return "unknown"
}

// getCurrentActiveModel tries to determine the current active model from session data
func getCurrentActiveModel() string {
	sessionData := fetchCurrentSessionData()
	if sessionData == nil {
		return ""
	}

	currentSession := findCurrentWorkingDirSession(sessionData.Sessions)
	if currentSession == nil {
		return ""
	}

	return determineActiveModel(currentSession)
}

// findCurrentWorkingDirSession finds the session for current working directory
func findCurrentWorkingDirSession(sessions []SessionInfo) *SessionInfo {
	currentDir := getCurrentWorkingDir()
	for i := range sessions {
		if strings.Contains(sessions[i].SessionID, currentDir) {
			return &sessions[i]
		}
	}
	return nil
}

// determineActiveModel determines the active model from session data
func determineActiveModel(session *SessionInfo) string {
	// If only one model is used in this session, that's the current one
	if len(session.ModelsUsed) == 1 {
		return formatModelName(session.ModelsUsed[0])
	}

	if len(session.ModelBreakdowns) == 0 {
		return ""
	}

	// Try intelligent heuristics for multiple models
	return selectModelFromBreakdowns(session.ModelBreakdowns)
}

// selectModelFromBreakdowns selects the most likely active model
func selectModelFromBreakdowns(breakdowns []ModelBreakdown) string {
	opusBreakdown, sonnetBreakdown := categorizeModelBreakdowns(breakdowns)

	// If both models are present, prefer Sonnet as it's likely the current one
	if sonnetBreakdown != nil && opusBreakdown != nil {
		return sonnetBreakdown.ModelName
	}

	// If only one major model, use it
	if sonnetBreakdown != nil {
		return sonnetBreakdown.ModelName
	}
	if opusBreakdown != nil {
		return opusBreakdown.ModelName
	}

	// Fallback to highest output tokens
	return selectModelWithHighestTokens(breakdowns)
}

// categorizeModelBreakdowns categorizes models into Opus and Sonnet
func categorizeModelBreakdowns(breakdowns []ModelBreakdown) (opus, sonnet *ModelBreakdown) {
	for i := range breakdowns {
		breakdown := &breakdowns[i]
		modelLower := strings.ToLower(breakdown.ModelName)
		if strings.Contains(modelLower, "opus") {
			opus = breakdown
		} else if strings.Contains(modelLower, "sonnet") {
			sonnet = breakdown
		}
	}
	return
}

// selectModelWithHighestTokens selects model with highest output tokens
func selectModelWithHighestTokens(breakdowns []ModelBreakdown) string {
	var maxOutputs int
	var currentModel string
	for _, breakdown := range breakdowns {
		if breakdown.OutputTokens > maxOutputs {
			maxOutputs = breakdown.OutputTokens
			currentModel = breakdown.ModelName
		}
	}
	if currentModel != "" {
		return formatModelName(currentModel)
	}
	return ""
}

// formatModelName converts full model name to display name
func formatModelName(fullName string) string {
	// Use the actual model name from ccusage as-is
	return fullName
}
