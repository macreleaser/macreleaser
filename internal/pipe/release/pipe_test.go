package release

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
				Release: config.ReleaseConfig{
					GitHub: config.GitHubConfig{
						Owner: "testuser",
						Repo:  "testrepo",
						Draft: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with draft",
			config: &config.Config{
				Release: config.ReleaseConfig{
					GitHub: config.GitHubConfig{
						Owner: "testuser",
						Repo:  "testrepo",
						Draft: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing owner",
			config: &config.Config{
				Release: config.ReleaseConfig{
					GitHub: config.GitHubConfig{
						Owner: "",
						Repo:  "testrepo",
						Draft: false,
					},
				},
			},
			wantErr: true,
			errMsg:  "release.github.owner is required",
		},
		{
			name: "missing repo",
			config: &config.Config{
				Release: config.ReleaseConfig{
					GitHub: config.GitHubConfig{
						Owner: "testuser",
						Repo:  "",
						Draft: false,
					},
				},
			},
			wantErr: true,
			errMsg:  "release.github.repo is required",
		},
		{
			name: "both fields missing",
			config: &config.Config{
				Release: config.ReleaseConfig{
					GitHub: config.GitHubConfig{
						Owner: "",
						Repo:  "",
						Draft: false,
					},
				},
			},
			wantErr: true,
			errMsg:  "release.github.owner is required",
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
	expected := "validating release configuration"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}
