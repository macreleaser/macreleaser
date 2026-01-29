package homebrew

import (
	"github.com/macreleaser/macreleaser/pkg/config"
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// Pipe validates homebrew configuration
type Pipe struct{}

func (Pipe) String() string { return "validating homebrew configuration" }

func (Pipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Homebrew

	if err := validate.RequiredString(cfg.Cask.Name, "homebrew.cask.name"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.Cask.Desc, "homebrew.cask.desc"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.Cask.Homepage, "homebrew.cask.homepage"); err != nil {
		return err
	}

	// If custom tap is configured, validate its required fields
	if isTapConfigured(cfg.Tap) {
		if err := validate.RequiredString(cfg.Tap.Owner, "homebrew.tap.owner"); err != nil {
			return err
		}
		if err := validate.RequiredString(cfg.Tap.Name, "homebrew.tap.name"); err != nil {
			return err
		}
		if err := validate.RequiredString(cfg.Tap.Token, "homebrew.tap.token"); err != nil {
			return err
		}
	}

	ctx.Logger.Debug("Homebrew configuration validated successfully")
	return nil
}

func isTapConfigured(cfg config.TapConfig) bool {
	return cfg.Owner != "" || cfg.Name != "" || cfg.Token != ""
}
