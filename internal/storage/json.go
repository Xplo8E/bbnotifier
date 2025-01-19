package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"bugbounty-notifier/internal/scraper"
)

type Storage struct {
	filePath string
	logger   *log.Logger
}

// NewStorage creates a new storage instance with logging
func NewStorage(filePath string) *Storage {
	// Create logger
	logger := log.New(os.Stdout, "[Storage] ", log.LstdFlags)

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	logger.Printf("Initialized storage with file: %s", filePath)
	return &Storage{
		filePath: filePath,
		logger:   logger,
	}
}

// LoadTargets reads the stored targets from the JSON file
func (s *Storage) LoadTargets() (*TargetHistory, error) {
	s.logger.Printf("Loading targets from %s", s.filePath)

	// If file doesn't exist or is empty, return empty history
	if info, err := os.Stat(s.filePath); os.IsNotExist(err) || info.Size() == 0 {
		s.logger.Printf("No existing targets file found or file is empty, creating new history")
		return &TargetHistory{
			LastUpdated: getISTTime(),
			Targets:     []StoredTarget{},
		}, nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read targets file: %w", err)
	}

	var history TargetHistory
	if err := json.Unmarshal(data, &history); err != nil {
		// If JSON parsing fails, return empty history instead of error
		s.logger.Printf("Failed to parse existing JSON, starting fresh: %v", err)
		return &TargetHistory{
			LastUpdated: getISTTime(),
			Targets:     []StoredTarget{},
		}, nil
	}

	s.logger.Printf("Successfully loaded %d targets", len(history.Targets))
	return &history, nil
}

// SaveTargets writes the targets to the JSON file
func (s *Storage) SaveTargets(history *TargetHistory) error {
	s.logger.Printf("Saving %d targets to file", len(history.Targets))

	// Ensure the directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal targets to JSON: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write targets file: %w", err)
	}

	s.logger.Printf("Successfully saved targets to %s", s.filePath)
	return nil
}

// CompareTargets compares new targets with stored ones and returns the differences
func (s *Storage) CompareTargets(newTargets []scraper.Target, storedHistory *TargetHistory) *TargetDiff {
	// Create maps for easier comparison
	storedMap := make(map[string]StoredTarget)
	for _, target := range storedHistory.Targets {
		key := target.ProgramName + "|" + target.Target
		storedMap[key] = target
	}

	newMap := make(map[string]scraper.Target)
	for _, target := range newTargets {
		key := target.ProgramName + "|" + target.Target
		newMap[key] = target
	}

	var diff TargetDiff

	// Find new targets
	for key, target := range newMap {
		if _, exists := storedMap[key]; !exists {
			diff.NewTargets = append(diff.NewTargets, StoredTarget{
				ProgramName: target.ProgramName,
				Target:      target.Target,
				Category:    target.Category,
				FirstSeen:   getISTTime(),
			})
		}
	}

	// Find removed targets
	for key, target := range storedMap {
		if _, exists := newMap[key]; !exists {
			diff.Removed = append(diff.Removed, target)
		}
	}

	return &diff
}

// UpdateTargets updates the stored targets with new ones
func (s *Storage) UpdateTargets(newTargets []scraper.Target) (*TargetDiff, error) {
	s.logger.Printf("Starting target update process with %d new targets", len(newTargets))

	// Filter out NO_IN_SCOPE_TABLE targets
	var filteredTargets []scraper.Target
	for _, target := range newTargets {
		if target.Target != "NO_IN_SCOPE_TABLE" {
			filteredTargets = append(filteredTargets, target)
		}
	}
	s.logger.Printf("Filtered out %d targets with NO_IN_SCOPE_TABLE", len(newTargets)-len(filteredTargets))

	history, err := s.LoadTargets()
	if err != nil {
		return nil, fmt.Errorf("failed to load existing targets: %w", err)
	}

	// Compare and get differences using filtered targets
	diff := s.CompareTargets(filteredTargets, history)
	s.logger.Printf("Found %d new targets and %d removed targets",
		len(diff.NewTargets), len(diff.Removed))

	// Handle new-targets.json file
	newTargetsPath := filepath.Join(filepath.Dir(s.filePath), "new-targets.json")
	if len(diff.NewTargets) > 0 {
		// Save new targets if found
		if err := s.SaveNewTargets(diff.NewTargets); err != nil {
			s.logger.Printf("Warning: Failed to save new-targets.json: %v", err)
		}
	} else {
		// Clear the new-targets.json file by creating an empty history
		emptyHistory := TargetHistory{
			LastUpdated: getISTTime(),
			Targets:     []StoredTarget{},
		}
		data, err := json.MarshalIndent(emptyHistory, "", "  ")
		if err != nil {
			s.logger.Printf("Warning: Failed to marshal empty history: %v", err)
		} else {
			if err := os.WriteFile(newTargetsPath, data, 0644); err != nil {
				s.logger.Printf("Warning: Failed to clear new-targets.json: %v", err)
			} else {
				s.logger.Printf("Cleared new-targets.json as no new targets were found")
			}
		}
	}

	// Create updated target list
	var updatedTargets []StoredTarget
	targetMap := make(map[string]bool)

	// Add existing targets that are still present
	for _, target := range filteredTargets {
		key := target.ProgramName + "|" + target.Target
		targetMap[key] = true

		found := false
		// Check if target already exists to preserve FirstSeen
		for _, stored := range history.Targets {
			if stored.ProgramName == target.ProgramName && stored.Target == target.Target {
				updatedTargets = append(updatedTargets, stored)
				found = true
				s.logger.Printf("Preserved existing target: %s - %s",
					stored.ProgramName, stored.Target)
				break
			}
		}

		// If it's a new target, add it with current timestamp
		if !found {
			newTarget := StoredTarget{
				ProgramName: target.ProgramName,
				Target:      target.Target,
				Category:    target.Category,
				FirstSeen:   getISTTime(),
			}
			updatedTargets = append(updatedTargets, newTarget)
			s.logger.Printf("Added new target: %s - %s",
				newTarget.ProgramName, newTarget.Target)
		}
	}

	// Update and save history
	history.Targets = updatedTargets
	history.LastUpdated = getISTTime()

	if err := s.SaveTargets(history); err != nil {
		return nil, fmt.Errorf("failed to save updated targets: %w", err)
	}

	return diff, nil
}

// SaveNewTargets saves new targets to a separate file
func (s *Storage) SaveNewTargets(newTargets []StoredTarget) error {
	s.logger.Printf("Saving %d new targets to new-targets.json", len(newTargets))

	// Create the new targets file path in the same directory as targets.json
	newTargetsPath := filepath.Join(filepath.Dir(s.filePath), "new-targets.json")

	// Create the structure to save
	newTargetsHistory := TargetHistory{
		LastUpdated: getISTTime(),
		Targets:     newTargets,
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(newTargetsHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal new targets to JSON: %w", err)
	}

	// Write to file (this will override any existing file)
	if err := os.WriteFile(newTargetsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write new-targets file: %w", err)
	}

	s.logger.Printf("Successfully saved new targets to %s", newTargetsPath)
	return nil
}

func getISTTime() time.Time {
	ist, _ := time.LoadLocation("Asia/Kolkata")
	return time.Now().In(ist)
}
