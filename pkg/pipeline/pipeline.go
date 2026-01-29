// Package pipeline executes all registered pipes in sequence.
//
// The pipeline orchestrates the release process by running each pipe
// in the order defined by the registry. If any pipe fails with an error
// (other than a SkipError), the pipeline stops and returns the error.
//
// Usage:
//
//	ctx := context.NewContext(context.Background(), cfg, logger)
//	if err := pipeline.Run(ctx); err != nil {
//	    // Handle error
//	}
package pipeline

import (
	"errors"
	"fmt"

	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/pipe"
)

// Run executes all registered pipes in sequence.
// Each pipe is run in the order defined in pkg/pipe/registry.go.
// If a pipe returns a SkipError, it is logged and the pipeline continues.
// If a pipe returns any other error, the pipeline stops and returns that error.
func Run(ctx *context.Context) error {
	for _, p := range pipe.All {
		ctx.Logger.Infof("Running: %s", p.String())

		if err := p.Run(ctx); err != nil {
			if isSkip(err) {
				ctx.Logger.Infof("Skipping: %v", err)
				continue
			}
			return fmt.Errorf("%s: %w", p.String(), err)
		}
	}
	return nil
}

func isSkip(err error) bool {
	var s pipe.IsSkip
	return errors.As(err, &s) && s.IsSkip()
}
