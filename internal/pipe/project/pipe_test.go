package project

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
				Project: config.ProjectConfig{
					Name:   "MyApp",
					Scheme: "MyApp",
				},
			},
			wantErr: false,
		},
		{
			name: "missing project name",
			config: &config.Config{
				Project: config.ProjectConfig{
					Name:   "",
					Scheme: "MyApp",
				},
			},
			wantErr: true,
			errMsg:  "project.name is required",
		},
		{
			name: "missing project scheme",
			config: &config.Config{
				Project: config.ProjectConfig{
					Name:   "MyApp",
					Scheme: "",
				},
			},
			wantErr: true,
			errMsg:  "project.scheme is required",
		},
		{
			name: "both fields missing",
			config: &config.Config{
				Project: config.ProjectConfig{
					Name:   "",
					Scheme: "",
				},
			},
			wantErr: true,
			errMsg:  "project.name is required",
		},
		{
			name: "valid with workspace",
			config: &config.Config{
				Project: config.ProjectConfig{
					Name:      "MyApp",
					Scheme:    "MyApp",
					Workspace: "MyApp.xcworkspace",
				},
			},
			wantErr: false,
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
	expected := "validating project configuration"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}
