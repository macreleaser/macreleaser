package notarize

import (
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// Pipe validates notarization configuration
type Pipe struct{}

func (Pipe) String() string { return "validating notarization configuration" }

func (Pipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Notarize

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
