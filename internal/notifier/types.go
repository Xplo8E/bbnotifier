package notifier

import "time"

// Target represents a bug bounty target for notification
type Target struct {
	ProgramName string
	Target      string
	Category    string
	FirstSeen   time.Time
}
