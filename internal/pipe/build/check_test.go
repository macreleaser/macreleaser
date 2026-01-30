package build

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
			name: "valid configuration with single arch",
			config: &config.Config{
				Build: config.BuildConfig{
					Configuration: "Release",
					Architectures: []string{"arm64"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with multiple archs",
			config: &config.Config{
				Build: config.BuildConfig{
					Configuration: "Release",
					Architectures: []string{"arm64", "x86_64"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with Universal Binary",
			config: &config.Config{
				Build: config.BuildConfig{
					Configuration: "Release",
					Architectures: []string{"Universal Binary"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing configuration",
			config: &config.Config{
				Build: config.BuildConfig{
					Configuration: "",
					Architectures: []string{"arm64"},
				},
			},
			wantErr: true,
			errMsg:  "build.configuration is required",
		},
		{
			name: "empty architectures",
			config: &config.Config{
				Build: config.BuildConfig{
					Configuration: "Release",
					Architectures: []string{},
				},
			},
			wantErr: true,
			errMsg:  "build.architectures requires at least one item",
		},
		{
			name: "nil architectures",
			config: &config.Config{
				Build: config.BuildConfig{
					Configuration: "Release",
					Architectures: nil,
				},
			},
			wantErr: true,
			errMsg:  "build.architectures requires at least one item",
		},
		{
			name: "invalid architecture",
			config: &config.Config{
				Build: config.BuildConfig{
					Configuration: "Release",
					Architectures: []string{"invalid-arch"},
				},
			},
			wantErr: true,
			errMsg:  "invalid build.architectures: invalid-arch",
		},
		{
			name: "mixed valid and invalid architectures",
			config: &config.Config{
				Build: config.BuildConfig{
					Configuration: "Release",
					Architectures: []string{"arm64", "invalid", "x86_64"},
				},
			},
			wantErr: true,
			errMsg:  "invalid build.architectures: invalid",
		},
		{
			name: "all valid architectures",
			config: &config.Config{
				Build: config.BuildConfig{
					Configuration: "Debug",
					Architectures: []string{"arm64", "x86_64", "Universal Binary"},
				},
			},
			wantErr: false,
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

func TestCheckPipeString(t *testing.T) {
	p := CheckPipe{}
	expected := "validating build configuration"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}
