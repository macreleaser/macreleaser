package notarize

import (
	"context"
	"strings"
	"testing"

	"github.com/macreleaser/macreleaser/pkg/config"
	ctx "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/sirupsen/logrus"
)

func TestPipe(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			config: &config.Config{
				Notarize: config.NotarizeConfig{
					AppleID:  "test@example.com",
					TeamID:   "TEAM123",
					Password: "env(APPLE_PASSWORD)",
				},
			},
			wantErr: false,
		},
		{
			name: "missing apple_id",
			config: &config.Config{
				Notarize: config.NotarizeConfig{
					AppleID:  "",
					TeamID:   "TEAM123",
					Password: "env(APPLE_PASSWORD)",
				},
			},
			wantErr: true,
			errMsg:  "notarize.apple_id is required",
		},
		{
			name: "missing team_id",
			config: &config.Config{
				Notarize: config.NotarizeConfig{
					AppleID:  "test@example.com",
					TeamID:   "",
					Password: "env(APPLE_PASSWORD)",
				},
			},
			wantErr: true,
			errMsg:  "notarize.team_id is required",
		},
		{
			name: "missing password",
			config: &config.Config{
				Notarize: config.NotarizeConfig{
					AppleID:  "test@example.com",
					TeamID:   "TEAM123",
					Password: "",
				},
			},
			wantErr: true,
			errMsg:  "notarize.password is required",
		},
		{
			name: "all fields missing",
			config: &config.Config{
				Notarize: config.NotarizeConfig{
					AppleID:  "",
					TeamID:   "",
					Password: "",
				},
			},
			wantErr: true,
			errMsg:  "notarize.apple_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := ctx.NewContext(context.Background(), tt.config, logger)
			err := Pipe{}.Run(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Run() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestPipeString(t *testing.T) {
	p := Pipe{}
	expected := "validating notarization configuration"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}
