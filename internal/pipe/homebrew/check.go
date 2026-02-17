package homebrew

import (
	"github.com/macreleaser/macreleaser/pkg/config"
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/env"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// CheckPipe validates homebrew configuration
type CheckPipe struct{}

func (CheckPipe) String() string { return "validating homebrew configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
	if ctx.SkipPublish {
		return skipError("homebrew publishing skipped")
	}

	cfg := ctx.Config.Homebrew

	if err := env.CheckResolved(cfg.Cask.Name, "homebrew.cask.name"); err != nil {
		return err
	}
	if err := env.CheckResolved(cfg.Cask.Desc, "homebrew.cask.desc"); err != nil {
		return err
	}
	if err := env.CheckResolved(cfg.Cask.Homepage, "homebrew.cask.homepage"); err != nil {
		return err
	}

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
		if err := env.CheckResolved(cfg.Tap.Owner, "homebrew.tap.owner"); err != nil {
			return err
		}
		if err := env.CheckResolved(cfg.Tap.Name, "homebrew.tap.name"); err != nil {
			return err
		}
		if err := env.CheckResolved(cfg.Tap.Token, "homebrew.tap.token"); err != nil {
			return err
		}

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
