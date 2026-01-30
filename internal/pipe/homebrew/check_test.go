package homebrew

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
			name: "valid cask configuration only",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "myapp",
						Desc:     "My awesome macOS application",
						Homepage: "https://github.com/user/myapp",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with custom tap",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "myapp",
						Desc:     "My awesome macOS application",
						Homepage: "https://github.com/user/myapp",
					},
					Tap: config.TapConfig{
						Owner: "user",
						Name:  "homebrew-tap",
						Token: "env(HOMEBREW_TAP_TOKEN)",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with official tap",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "myapp",
						Desc:     "My awesome macOS application",
						Homepage: "https://github.com/user/myapp",
					},
					Official: config.OfficialConfig{
						Enabled: true,
						Token:   "env(HOMEBREW_OFFICIAL_TOKEN)",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing cask name",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "",
						Desc:     "My awesome macOS application",
						Homepage: "https://github.com/user/myapp",
					},
				},
			},
			wantErr: true,
			errMsg:  "homebrew.cask.name is required",
		},
		{
			name: "missing cask description",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "myapp",
						Desc:     "",
						Homepage: "https://github.com/user/myapp",
					},
				},
			},
			wantErr: true,
			errMsg:  "homebrew.cask.desc is required",
		},
		{
			name: "missing cask homepage",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "myapp",
						Desc:     "My awesome macOS application",
						Homepage: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "homebrew.cask.homepage is required",
		},
		{
			name: "custom tap with missing owner",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "myapp",
						Desc:     "My awesome macOS application",
						Homepage: "https://github.com/user/myapp",
					},
					Tap: config.TapConfig{
						Owner: "",
						Name:  "homebrew-tap",
						Token: "env(HOMEBREW_TAP_TOKEN)",
					},
				},
			},
			wantErr: true,
			errMsg:  "homebrew.tap.owner is required",
		},
		{
			name: "custom tap with missing name",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "myapp",
						Desc:     "My awesome macOS application",
						Homepage: "https://github.com/user/myapp",
					},
					Tap: config.TapConfig{
						Owner: "user",
						Name:  "",
						Token: "env(HOMEBREW_TAP_TOKEN)",
					},
				},
			},
			wantErr: true,
			errMsg:  "homebrew.tap.name is required",
		},
		{
			name: "custom tap with missing token",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "myapp",
						Desc:     "My awesome macOS application",
						Homepage: "https://github.com/user/myapp",
					},
					Tap: config.TapConfig{
						Owner: "user",
						Name:  "homebrew-tap",
						Token: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "homebrew.tap.token is required",
		},
		{
			name: "partial tap configuration triggers validation",
			config: &config.Config{
				Homebrew: config.HomebrewConfig{
					Cask: config.CaskConfig{
						Name:     "myapp",
						Desc:     "My awesome macOS application",
						Homepage: "https://github.com/user/myapp",
					},
					Tap: config.TapConfig{
						Owner: "user",
						Name:  "",
						Token: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "homebrew.tap.name is required",
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
	expected := "validating homebrew configuration"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}
