package archive

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
			name: "valid configuration with single format",
			config: &config.Config{
				Archive: config.ArchiveConfig{
					Formats: []string{"dmg"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with multiple formats",
			config: &config.Config{
				Archive: config.ArchiveConfig{
					Formats: []string{"dmg", "zip"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with all formats",
			config: &config.Config{
				Archive: config.ArchiveConfig{
					Formats: []string{"dmg", "zip", "app"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty formats",
			config: &config.Config{
				Archive: config.ArchiveConfig{
					Formats: []string{},
				},
			},
			wantErr: true,
			errMsg:  "archive.formats requires at least one item",
		},
		{
			name: "nil formats",
			config: &config.Config{
				Archive: config.ArchiveConfig{
					Formats: nil,
				},
			},
			wantErr: true,
			errMsg:  "archive.formats requires at least one item",
		},
		{
			name: "invalid format",
			config: &config.Config{
				Archive: config.ArchiveConfig{
					Formats: []string{"tar"},
				},
			},
			wantErr: true,
			errMsg:  "invalid archive.formats: tar",
		},
		{
			name: "mixed valid and invalid formats",
			config: &config.Config{
				Archive: config.ArchiveConfig{
					Formats: []string{"dmg", "invalid", "zip"},
				},
			},
			wantErr: true,
			errMsg:  "invalid archive.formats: invalid",
		},
		{
			name: "uppercase format",
			config: &config.Config{
				Archive: config.ArchiveConfig{
					Formats: []string{"DMG"},
				},
			},
			wantErr: true,
			errMsg:  "invalid archive.formats: DMG",
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
	expected := "validating archive configuration"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}
