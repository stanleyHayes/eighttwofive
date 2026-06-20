// Package config loads and validates runtime configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	errMongoURIRequired = errors.New("MONGODB_URI is required")
	errInvalidEnv       = errors.New("ENV must be development or production")
)

// Config is the complete runtime configuration for the API.
type Config struct {
	Env                string
	Port               string
	MongoURI           string
	MongoDB            string
	WebURL             string
	AdminEmails        []string
	CORSAllowedOrigins []string

	ResendAPIKey string
	EmailFrom    string

	PaystackSecretKey string

	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
}

// IsProduction reports whether the API runs in production mode.
func (c *Config) IsProduction() bool { return c.Env == "production" }

// EmailEnabled reports whether a Resend API key is configured.
func (c *Config) EmailEnabled() bool { return c.ResendAPIKey != "" }

// Warnings returns non-fatal configuration problems worth surfacing at boot.
// These are setups that look fine but silently fail to reach customers — most
// often an email sender that Resend will refuse to deliver to anyone but the
// account owner.
func (c *Config) Warnings() []string {
	var out []string

	if c.IsProduction() && !c.EmailEnabled() {
		out = append(out,
			"RESEND_API_KEY is not set: sign-in links and emails are only logged, "+
				"so customers cannot receive them — set RESEND_API_KEY and EMAIL_FROM")
	}

	if c.EmailEnabled() && c.usesSharedResendSender() {
		out = append(out,
			"EMAIL_FROM uses Resend's shared onboarding@resend.dev sender, which only "+
				"delivers to your own Resend account email; verify a domain in Resend and "+
				"set EMAIL_FROM to an address on it so customers receive emails")
	}

	return out
}

// PaystackEnabled reports whether a Paystack secret key is configured.
func (c *Config) PaystackEnabled() bool { return c.PaystackSecretKey != "" }

// UploadsEnabled reports whether the full Cloudinary credential trio is configured.
func (c *Config) UploadsEnabled() bool {
	return c.CloudinaryCloudName != "" && c.CloudinaryAPIKey != "" && c.CloudinaryAPISecret != ""
}

// usesSharedResendSender reports whether EMAIL_FROM is Resend's shared test
// sender (anything @resend.dev). Resend only delivers from that address to the
// account owner's own email, so customers never receive anything.
func (c *Config) usesSharedResendSender() bool {
	return strings.Contains(strings.ToLower(c.EmailFrom), "resend.dev")
}

// Load reads configuration from environment variables and validates it.
func Load() (*Config, error) {
	cfg := &Config{
		Env:                 getEnv("ENV", "development"),
		Port:                getEnv("PORT", "8080"),
		MongoURI:            os.Getenv("MONGODB_URI"),
		MongoDB:             getEnv("MONGODB_DB", "eightfivetwo"),
		WebURL:              getEnv("WEB_URL", "http://localhost:5173"),
		AdminEmails:         splitAndTrim(os.Getenv("ADMIN_EMAILS")),
		CORSAllowedOrigins:  splitAndTrim(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		ResendAPIKey:        os.Getenv("RESEND_API_KEY"),
		EmailFrom:           getEnv("EMAIL_FROM", "Eight Two Five <onboarding@resend.dev>"),
		PaystackSecretKey:   os.Getenv("PAYSTACK_SECRET_KEY"),
		CloudinaryCloudName: os.Getenv("CLOUDINARY_CLOUD_NAME"),
		CloudinaryAPIKey:    os.Getenv("CLOUDINARY_API_KEY"),
		CloudinaryAPISecret: os.Getenv("CLOUDINARY_API_SECRET"),
	}

	if cfg.MongoURI == "" {
		return nil, errMongoURIRequired
	}

	if cfg.Env != "development" && cfg.Env != "production" {
		return nil, fmt.Errorf("%w, got %q", errInvalidEnv, cfg.Env)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")

	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}

	return out
}
