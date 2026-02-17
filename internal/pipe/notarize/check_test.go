package notarize

import (
	"context"
	"strings"
	"testing"

	"github.com/macreleaser/macreleaser/pkg/config"
	macCtx "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/sirupsen/logrus"
)

func TestCheckPipe(t *testing.T) {
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
					Password: "xxxx-xxxx-xxxx-xxxx",
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
					Password: "xxxx-xxxx-xxxx-xxxx",
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
					Password: "xxxx-xxxx-xxxx-xxxx",
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
			ctx := macCtx.NewContext(context.Background(), tt.config, logger)
			err := CheckPipe{}.Run(ctx)

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

func TestCheckPipeSkipNotarize(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Config with missing fields — would normally fail validation
	ctx := macCtx.NewContext(context.Background(), &config.Config{}, logger)
	ctx.SkipNotarize = true

	err := CheckPipe{}.Run(ctx)
	if err == nil {
		t.Fatal("expected skipError, got nil")
	}

	if !strings.Contains(err.Error(), "skipped") {
		t.Errorf("error = %v, want containing 'skipped'", err)
	}
}

func TestCheckPipeString(t *testing.T) {
	p := CheckPipe{}
	expected := "validating notarization configuration"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}
