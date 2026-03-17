package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	PollInterval   time.Duration   `yaml:"poll_interval"`
	Database       DatabaseConfig  `yaml:"database"`
	Mailboxes      []MailboxConfig `yaml:"mailboxes"`
	AllowedSenders []string        `yaml:"allowed_senders"` // Only process emails from these senders
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	DSN            string `yaml:"dsn"`
	SSLRootCert    string `yaml:"ssl_root_cert"`
	MaxOpenConns   int    `yaml:"max_open_conns"`
	MaxIdleConns   int    `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// MailboxConfig represents mailbox configuration
type MailboxConfig struct {
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Parse allowed_senders - support both array and comma-separated string
	if len(cfg.AllowedSenders) == 1 && strings.Contains(cfg.AllowedSenders[0], ",") {
		// Split comma-separated string into array
		parts := strings.Split(cfg.AllowedSenders[0], ",")
		cfg.AllowedSenders = make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				cfg.AllowedSenders = append(cfg.AllowedSenders, trimmed)
			}
		}
	}

	// Set defaults
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 25
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 5 * time.Minute
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.PollInterval <= 0 {
		return fmt.Errorf("poll_interval must be greater than 0")
	}

	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}

	if len(c.Mailboxes) == 0 {
		return fmt.Errorf("at least one mailbox must be configured")
	}

	for i, mb := range c.Mailboxes {
		if mb.Email == "" {
			return fmt.Errorf("mailboxes[%d].email is required", i)
		}
		if mb.Password == "" {
			return fmt.Errorf("mailboxes[%d].password is required", i)
		}
		if mb.Host == "" {
			return fmt.Errorf("mailboxes[%d].host is required", i)
		}
		if mb.Port == 0 {
			return fmt.Errorf("mailboxes[%d].port is required", i)
		}
	}

	return nil
}
