package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// JSONLMessage represents a message entry in the JSONL file
type JSONLMessage struct {
	SessionID string           `json:"sessionId"`
	Type      string           `json:"type"`
	UUID      string           `json:"uuid"`
	Message   AssistantMessage `json:"message"`
}

// AssistantMessage represents the message field in JSONL
type AssistantMessage struct {
	Role  string     `json:"role"`
	Usage TokenUsage `json:"usage"`
}

// TokenUsage represents token usage in a message
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// MessageTokenReader reads token data from JSONL files
type MessageTokenReader struct {
	claudeProjectsDir string
}

// NewMessageTokenReader creates a new reader
func NewMessageTokenReader() *MessageTokenReader {
	homeDir, _ := os.UserHomeDir()
	claudeProjectsDir := filepath.Join(homeDir, ".config", "claude", "projects")

	return &MessageTokenReader{
		claudeProjectsDir: claudeProjectsDir,
	}
}

// GetBlockTokens retrieves all message tokens for a specific time range across all projects
func (r *MessageTokenReader) GetBlockTokens(startTime, endTime string) ([]int, error) {
	// Get all project directories
	projectDirs, err := r.getAllProjectDirs()
	if err != nil {
		return nil, err
	}

	var allTokens []int

	// Search through all project directories
	for _, projectDir := range projectDirs {
		// Find JSONL files in this project
		files, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
		if err != nil {
			continue // Skip this project on error
		}

		// Read tokens from each file
		for _, file := range files {
			tokens, err := r.readBlockTokensFromFile(file, startTime, endTime)
			if err != nil {
				continue // Skip files with errors
			}
			allTokens = append(allTokens, tokens...)
		}
	}

	return allTokens, nil
}

// getAllProjectDirs returns all project directories under ~/.config/claude/projects/
func (r *MessageTokenReader) getAllProjectDirs() ([]string, error) {
	entries, err := os.ReadDir(r.claudeProjectsDir)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(r.claudeProjectsDir, entry.Name()))
		}
	}

	return dirs, nil
}

// readBlockTokensFromFile reads tokens for messages within a time range from a file
func (r *MessageTokenReader) readBlockTokensFromFile(filename, startTime, endTime string) ([]int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tokens []int
	scanner := bufio.NewScanner(file)

	// Parse time boundaries
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return nil, err
	}

	for scanner.Scan() {
		var msg struct {
			Timestamp string           `json:"timestamp"`
			Type      string           `json:"type"`
			Message   AssistantMessage `json:"message"`
		}

		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue // Skip malformed lines
		}

		// Only process assistant messages
		if msg.Type != "assistant" {
			continue
		}

		// Check if message is within time range
		msgTime, err := time.Parse(time.RFC3339, msg.Timestamp)
		if err != nil {
			continue
		}

		// Check if message is within time range (inclusive)
		if (msgTime.Equal(start) || msgTime.After(start)) && (msgTime.Before(end) || msgTime.Equal(end)) {
			totalTokens := msg.Message.Usage.InputTokens + msg.Message.Usage.OutputTokens
			if totalTokens > 0 {
				tokens = append(tokens, totalTokens)
			}
		}
	}

	return tokens, scanner.Err()
}

// CalculateMedianTokens calculates the median of token values
func CalculateMedianTokens(tokens []int) int {
	if len(tokens) == 0 {
		return 0
	}

	// Sort tokens
	sorted := make([]int, len(tokens))
	copy(sorted, tokens)
	sort.Ints(sorted)

	// Calculate median
	n := len(sorted)
	if n%2 == 0 {
		// Even number of elements: average of two middle values
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	// Odd number of elements: middle value
	return sorted[n/2]
}

// CalculateTrimmedMean calculates mean after removing top and bottom percentile
func CalculateTrimmedMean(tokens []int, trimPercent float64) int {
	if len(tokens) == 0 {
		return 0
	}

	// Sort tokens
	sorted := make([]int, len(tokens))
	copy(sorted, tokens)
	sort.Ints(sorted)

	// Calculate trim count
	trimCount := int(float64(len(sorted)) * trimPercent / 100.0)
	if trimCount == 0 && len(sorted) > 2 {
		trimCount = 1
	}

	// Trim from both ends
	if trimCount*2 >= len(sorted) {
		return sorted[len(sorted)/2] // Return median if trimming too much
	}

	trimmed := sorted[trimCount : len(sorted)-trimCount]

	// Calculate mean of trimmed data
	sum := 0
	for _, v := range trimmed {
		sum += v
	}

	return sum / len(trimmed)
}

// CalculateMode calculates the most frequent value
func CalculateMode(tokens []int) int {
	if len(tokens) == 0 {
		return 0
	}

	// Count frequencies
	freq := make(map[int]int)
	for _, v := range tokens {
		freq[v]++
	}

	// Find mode
	maxFreq := 0
	mode := 0
	for value, count := range freq {
		if count > maxFreq {
			maxFreq = count
			mode = value
		}
	}

	return mode
}
