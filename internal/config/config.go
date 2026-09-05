package config

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ProdBase    = "https://openapi.investec.com"
	SandboxBase = "https://openapisandbox.investec.com"
)

// Credentials are the three secrets needed to mint a token.
type Credentials struct {
	ClientID     string
	ClientSecret string
	APIKey       string
	// Source records where the secrets came from, for actionable errors.
	Source string
}

// Config is the resolved runtime configuration for a single invocation.
type Config struct {
	Base    string
	Sandbox bool
	Creds   Credentials
}

// fileConfig mirrors the user config.toml. Only credential fields are honored.
type fileConfig struct {
	ClientID          string
	ClientSecret      string
	APIKey            string
	CredentialCommand string
}

// Resolve builds a Config using precedence: env > user config.
// Secrets are never read from flags. Missing credentials are the caller's to
// surface as exit code 3.
func Resolve(sandbox, noInput bool) (*Config, error) {
	cfg := &Config{Sandbox: sandbox}
	if sandbox {
		cfg.Base = SandboxBase
	} else {
		cfg.Base = ProdBase
	}

	fc, err := loadFileConfig()
	if err != nil {
		return nil, err
	}

	creds := Credentials{Source: "none"}
	if fc != nil {
		if fc.CredentialCommand != "" {
			cc, err := runCredentialCommand(fc.CredentialCommand, noInput)
			if err != nil {
				return nil, err
			}
			creds = mergeCreds(creds, cc, "credential_command")
		}
		creds = mergeCreds(creds, Credentials{
			ClientID:     fc.ClientID,
			ClientSecret: fc.ClientSecret,
			APIKey:       fc.APIKey,
		}, "config.toml")
	}

	// Environment overrides config.
	creds = mergeCreds(creds, Credentials{
		ClientID:     os.Getenv("INVESTEC_CLIENT_ID"),
		ClientSecret: os.Getenv("INVESTEC_CLIENT_SECRET"),
		APIKey:       os.Getenv("INVESTEC_API_KEY"),
	}, "env")

	cfg.Creds = creds
	return cfg, nil
}

// Complete reports whether all three secrets are present.
func (c Credentials) Complete() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.APIKey != ""
}

// Missing lists the human-facing names of absent secrets.
func (c Credentials) Missing() []string {
	var m []string
	if c.ClientID == "" {
		m = append(m, "INVESTEC_CLIENT_ID (or client_id)")
	}
	if c.ClientSecret == "" {
		m = append(m, "INVESTEC_CLIENT_SECRET (or client_secret)")
	}
	if c.APIKey == "" {
		m = append(m, "INVESTEC_API_KEY (or api_key)")
	}
	return m
}

func mergeCreds(base, over Credentials, source string) Credentials {
	touched := false
	if over.ClientID != "" {
		base.ClientID = over.ClientID
		touched = true
	}
	if over.ClientSecret != "" {
		base.ClientSecret = over.ClientSecret
		touched = true
	}
	if over.APIKey != "" {
		base.APIKey = over.APIKey
		touched = true
	}
	if touched {
		base.Source = source
	}
	return base
}

func configDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "investec")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "investec")
}

func loadFileConfig() (*fileConfig, error) {
	path := filepath.Join(configDir(), "config.toml")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	return parseConfig(f)
}

// parseConfig is a deliberately tiny reader for the flat key = "value" schema
// this config uses. It avoids a third-party TOML dependency for a handful of
// string keys.
func parseConfig(f *os.File) (*fileConfig, error) {
	fc := &fileConfig{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"`)
		switch k {
		case "client_id":
			fc.ClientID = v
		case "client_secret":
			fc.ClientSecret = v
		case "api_key":
			fc.APIKey = v
		case "credential_command":
			fc.CredentialCommand = v
		}
	}
	return fc, sc.Err()
}

// runCredentialCommand executes an argv (no shell) that prints KEY=VALUE lines
// for any of client_id/client_secret/api_key. Under --no-input its stdin is
// closed so it cannot hang on a prompt.
func runCredentialCommand(command string, noInput bool) (Credentials, error) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return Credentials{}, fmt.Errorf("credential_command is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	if noInput {
		cmd.Stdin = nil
	}
	out, err := cmd.Output()
	if err != nil {
		return Credentials{}, fmt.Errorf("credential_command failed: %w", err)
	}
	if len(out) > 1<<20 {
		return Credentials{}, fmt.Errorf("credential_command output too large")
	}
	var c Credentials
	for _, line := range strings.Split(string(out), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "client_id", "INVESTEC_CLIENT_ID":
			c.ClientID = strings.TrimSpace(v)
		case "client_secret", "INVESTEC_CLIENT_SECRET":
			c.ClientSecret = strings.TrimSpace(v)
		case "api_key", "INVESTEC_API_KEY":
			c.APIKey = strings.TrimSpace(v)
		}
	}
	return c, nil
}
