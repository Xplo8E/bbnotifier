package scraper

import (
	"encoding/base64"
	"time"

	"github.com/sw33tLie/bbscope/pkg/platforms/hackerone"
)

// Target represents a bug bounty target with additional metadata
type Target struct {
	ProgramName string    `json:"program_name"`
	Target      string    `json:"target"`
	Category    string    `json:"category"`
	AddedAt     time.Time `json:"added_at"`
}

// Scraper handles the bug bounty program scraping
type Scraper struct {
	username string
	token    string
	config   Config
}

// Config holds scraper configuration
type Config struct {
	BBPOnly       bool
	PrivateOnly   bool
	PublicOnly    bool
	PrintRealTime bool
	Active        bool
	IncludeOOS    bool
	Categories    string
	OutputFlags   string
	Delimiter     string
	Concurrency   int
}

// NewScraper creates a new scraper instance
func NewScraper(username, token string, config Config) *Scraper {
	return &Scraper{
		username: username,
		token:    token,
		config:   config,
	}
}

// GetH1Targets fetches all HackerOne targets
func (s *Scraper) GetH1Targets() ([]Target, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(s.username + ":" + s.token))

	bbpOnly := s.config.BBPOnly
	pvtOnly := s.config.PrivateOnly
	publicOnly := s.config.PublicOnly
	categories := s.config.Categories
	active := s.config.Active
	concurrency := s.config.Concurrency
	printRealTime := s.config.PrintRealTime
	outputFlags := s.config.OutputFlags
	delimiter := s.config.Delimiter
	includeOOS := s.config.IncludeOOS

	scope, err := hackerone.GetAllProgramsScope(auth, bbpOnly, pvtOnly, publicOnly, categories, active, concurrency, printRealTime, outputFlags, delimiter, includeOOS)
	if err != nil {
		return nil, err
	}

	var targets []Target
	now := time.Now()

	for _, program := range scope {
		for _, elem := range program.InScope {
			targets = append(targets, Target{
				ProgramName: program.Url,
				Target:      elem.Target,
				Category:    elem.Category,
				AddedAt:     now,
			})
		}
	}

	return targets, nil
}
