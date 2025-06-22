package main

import (
	"time"
)

// Session represents an active Claude session with all related data
type Session struct {
	Block     *Block
	StartTime time.Time
	EndTime   time.Time
	Metrics   SessionMetrics
	BurnRate  float64
	TodayCost float64
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
		Block:     block,
		StartTime: startTime,
		EndTime:   endTime,
		BurnRate:  calculateHourlyBurnRate(allBlocks, currentTime),
		TodayCost: fetchTodayTotalCost(currentTime),
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
