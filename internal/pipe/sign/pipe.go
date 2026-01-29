package sign

import (
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// Pipe validates signing configuration
type Pipe struct{}

func (Pipe) String() string { return "validating signing configuration" }

func (Pipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Sign

	if err := validate.RequiredString(cfg.Identity, "sign.identity"); err != nil {
		return err
	}

	ctx.Logger.Debug("Signing configuration validated successfully")
	return nil
}
