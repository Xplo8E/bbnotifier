package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"bugbounty-notifier/internal/config"
	"bugbounty-notifier/internal/notifier"
	"bugbounty-notifier/internal/scraper"
	"bugbounty-notifier/internal/storage"
)

func main() {
	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting bug bounty target scanner")

	// Load configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize storage
	store := storage.NewStorage(filepath.Join(cfg.App.DataDir, "targets.json"))

	// Initialize scraper with config
	h1Scraper := scraper.NewScraper(
		cfg.Credentials.H1Username,
		cfg.Credentials.H1Token,
		scraper.Config{
			BBPOnly:       cfg.Scraper.BBPOnly,
			PrivateOnly:   cfg.Scraper.PrivateOnly,
			PublicOnly:    cfg.Scraper.PublicOnly,
			Categories:    cfg.Scraper.Categories,
			Active:        cfg.Scraper.Active,
			IncludeOOS:    cfg.Scraper.IncludeOOS,
			PrintRealTime: cfg.Scraper.PrintRealTime,
			Concurrency:   cfg.App.Concurrency,

			// bbpOnly := s.config.BBPOnly
			// pvtOnly := s.config.PrivateOnly
			// publicOnly := s.config.PublicOnly
			// categories := s.config.Categories
			// active := s.config.Active
			// concurrency := s.config.Concurrency
			// printRealTime := s.config.PrintRealTime
			// outputFlags := s.config.OutputFlags
			// delimiter := s.config.Delimiter
			// includeOOS := s.config.IncludeOOS
		},
	)

	// Fetch new targets
	log.Println("Fetching targets from HackerOne...")
	startTime := time.Now()
	targets, err := h1Scraper.GetH1Targets()
	if err != nil {
		log.Fatalf("Failed to get targets: %v", err)
	} else {
		fmt.Println("Found targets")
	}
	log.Printf("Fetched %d targets in %v", len(targets), time.Since(startTime))

	// Update storage and get differences
	log.Println("Updating storage with new targets...")
	diff, err := store.UpdateTargets(targets)
	if err != nil {
		log.Fatalf("Failed to update targets: %v", err)
	}

	// Detailed logging of changes
	if len(diff.NewTargets) > 0 {
		log.Printf("Found %d new targets:", len(diff.NewTargets))
		for _, target := range diff.NewTargets {
			log.Printf("  - [%s] %s (%s)",
				target.ProgramName, target.Target, target.Category)
		}
	} else {
		log.Println("No new targets found")
	}

	if len(diff.Removed) > 0 {
		log.Printf("Found %d removed targets:", len(diff.Removed))
		for _, target := range diff.Removed {
			log.Printf("  - [%s] %s (%s)",
				target.ProgramName, target.Target, target.Category)
		}
	}

	log.Println("Scan completed successfully")

	// Notification handling
	if len(diff.NewTargets) > 0 && cfg.Notifications.Enabled {
		if cfg.Credentials.SlackWebhook == "" {
			log.Println("Slack webhook URL not configured, skipping notification")
		} else {
			n := notifier.NewSlackNotifier(cfg.Credentials.SlackWebhook)
			// Convert storage.StoredTarget to notifier.Target
			var notifyTargets []notifier.Target
			for _, t := range diff.NewTargets {
				notifyTargets = append(notifyTargets, notifier.Target{
					ProgramName: t.ProgramName,
					Target:      t.Target,
					Category:    t.Category,
					FirstSeen:   t.FirstSeen,
				})
			}

			err = n.NotifyNewTargets(notifyTargets)
			if err != nil {
				log.Printf("Failed to send Slack notification: %v", err)
			}
		}
	}
}

// TODO: Initialize components
// - Load configuration - DONE
// - Setup scraper - DONE
// - Setup notifier - Working on it
// - Setup storage - DONE

// TODO: Run the scan
// - Fetch  targets - DONE
// - Compare with stored targets - DONE
// - Update storage - DONE
// - Send notifications - Working on it
