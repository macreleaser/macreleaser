package notarize

import (
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/env"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// CheckPipe validates notarization configuration
type CheckPipe struct{}

func (CheckPipe) String() string { return "validating notarization configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
	if ctx.SkipNotarize {
		return skipError("notarization skipped via --skip-notarize")
	}

	cfg := ctx.Config.Notarize

	if err := env.CheckResolved(cfg.AppleID, "notarize.apple_id"); err != nil {
		return err
	}
	if err := env.CheckResolved(cfg.TeamID, "notarize.team_id"); err != nil {
		return err
	}
	if err := env.CheckResolved(cfg.Password, "notarize.password"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.AppleID, "notarize.apple_id"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.TeamID, "notarize.team_id"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.Password, "notarize.password"); err != nil {
		return err
	}

	ctx.Logger.Debug("Notarization configuration validated successfully")
	return nil
}
