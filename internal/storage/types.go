package storage

import (
	"time"
)

// TargetHistory represents the historical data of targets
type TargetHistory struct {
	LastUpdated time.Time        `json:"last_updated"`
	Targets     []StoredTarget   `json:"targets"`
}

// StoredTarget represents a target with its discovery timestamp
type StoredTarget struct {
	ProgramName string    `json:"program_name"`
	Target      string    `json:"target"`
	Category    string    `json:"category"`
	FirstSeen   time.Time `json:"first_seen"`
}

// TargetDiff represents the difference between two scans
type TargetDiff struct {
	NewTargets []StoredTarget
	Removed    []StoredTarget
}