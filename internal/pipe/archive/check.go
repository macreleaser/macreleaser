package archive

import (
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// CheckPipe validates archive configuration
type CheckPipe struct{}

func (CheckPipe) String() string { return "validating archive configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Archive

	if err := validate.RequiredSlice(cfg.Formats, "archive.formats"); err != nil {
		return err
	}

	validFormats := []string{"dmg", "zip", "app"}
	if err := validate.AllOneOf(cfg.Formats, validFormats, "archive.formats"); err != nil {
		return err
	}

	ctx.Logger.Debug("Archive configuration validated successfully")
	return nil
}
