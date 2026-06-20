package config_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/config"
)

func TestConfig_Warnings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		cfg         config.Config
		wantContain string // substring expected in some warning, or "" for none
	}{
		{
			name:        "production without a Resend key warns customers can't get email",
			cfg:         config.Config{Env: "production"},
			wantContain: "RESEND_API_KEY is not set",
		},
		{
			name:        "shared resend.dev sender warns it only reaches the owner",
			cfg:         config.Config{Env: "production", ResendAPIKey: "re_x", EmailFrom: "Brand <onboarding@resend.dev>"},
			wantContain: "shared onboarding@resend.dev",
		},
		{
			name:        "a verified custom domain in production is clean",
			cfg:         config.Config{Env: "production", ResendAPIKey: "re_x", EmailFrom: "Brand <hello@eighttwofive.com>"},
			wantContain: "",
		},
		{
			name:        "development with no key is fine (sends are logged on purpose)",
			cfg:         config.Config{Env: "development", EmailFrom: "Brand <onboarding@resend.dev>"},
			wantContain: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			warnings := tc.cfg.Warnings()

			if tc.wantContain == "" {
				assert.Empty(t, warnings)

				return
			}

			assert.Contains(t, strings.Join(warnings, "\n"), tc.wantContain)
		})
	}
}
