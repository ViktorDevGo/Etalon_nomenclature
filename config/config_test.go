package config

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				PollInterval: 1 * time.Minute,
				Database: DatabaseConfig{
					DSN: "postgresql://localhost/test",
				},
				Mailboxes: []MailboxConfig{
					{
						Email:    "test@example.com",
						Password: "password",
						Host:     "mail.example.com",
						Port:     993,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing poll_interval",
			config: &Config{
				PollInterval: 0,
				Database: DatabaseConfig{
					DSN: "postgresql://localhost/test",
				},
				Mailboxes: []MailboxConfig{
					{
						Email:    "test@example.com",
						Password: "password",
						Host:     "mail.example.com",
						Port:     993,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing database DSN",
			config: &Config{
				PollInterval: 1 * time.Minute,
				Database: DatabaseConfig{
					DSN: "",
				},
				Mailboxes: []MailboxConfig{
					{
						Email:    "test@example.com",
						Password: "password",
						Host:     "mail.example.com",
						Port:     993,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no mailboxes",
			config: &Config{
				PollInterval: 1 * time.Minute,
				Database: DatabaseConfig{
					DSN: "postgresql://localhost/test",
				},
				Mailboxes: []MailboxConfig{},
			},
			wantErr: true,
		},
		{
			name: "mailbox missing email",
			config: &Config{
				PollInterval: 1 * time.Minute,
				Database: DatabaseConfig{
					DSN: "postgresql://localhost/test",
				},
				Mailboxes: []MailboxConfig{
					{
						Email:    "",
						Password: "password",
						Host:     "mail.example.com",
						Port:     993,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
