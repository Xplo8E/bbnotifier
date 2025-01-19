package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	App struct {
		Concurrency int    `yaml:"concurrency"`
		DataDir     string `yaml:"data_dir"`
	} `yaml:"app"`

	Scraper struct {
		BBPOnly       bool   `yaml:"bbp_only"`
		PrivateOnly   bool   `yaml:"private_only"`
		PublicOnly    bool   `yaml:"public_only"`
		Categories    string `yaml:"categories"`
		Active        bool   `yaml:"active"`
		IncludeOOS    bool   `yaml:"include_oos"`
		OutputFlags   string `yaml:"output_flags"`
		Delimiter     string `yaml:"delimiter"`
		PrintRealTime bool   `yaml:"print_real_time"`
	} `yaml:"scraper"`

	Notifications struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"notifications"`

	Credentials struct {
		H1Username   string `yaml:"h1_username"`
		H1Token      string `yaml:"h1_token"`
		SlackWebhook string `yaml:"slack_webhook"`
	} `yaml:"credentials"`
}

func LoadConfig(configPath string) (*Config, error) {
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Override with environment variables if present
	if envVar := os.Getenv("H1_USERNAME"); envVar != "" {
		config.Credentials.H1Username = envVar
	}
	if envVar := os.Getenv("H1_TOKEN"); envVar != "" {
		config.Credentials.H1Token = envVar
	}
	if envVar := os.Getenv("SLACK_WEBHOOK_URL"); envVar != "" {
		config.Credentials.SlackWebhook = envVar
	}
	if envVar := os.Getenv("SEND_NOTIFICATIONS"); envVar != "" {
		config.Notifications.Enabled = envVar == "true"
	}

	// Validate required fields
	if config.Credentials.H1Username == "" {
		return nil, fmt.Errorf("H1_USERNAME not set in config or environment")
	}
	if config.Credentials.H1Token == "" {
		return nil, fmt.Errorf("H1_TOKEN not set in config or environment")
	}

	return &config, nil
}
