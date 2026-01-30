// Package pipeline executes all registered pipes in sequence.
//
// The pipeline orchestrates the release process by running pipes in stages:
//   - Validation stage: runs all validation pipes to check configuration
//   - Execution stage: runs execution pipes to build, archive, and package
//
// Usage:
//
//	ctx := context.NewContext(context.Background(), cfg, logger)
//	if err := pipeline.RunValidation(ctx); err != nil {
//	    // Handle validation error
//	}
//	if err := pipeline.RunAll(ctx); err != nil {
//	    // Handle error
//	}
package pipeline

import (
	"errors"
	"fmt"
	"time"

	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/pipe"
)

// RunValidation executes only the validation pipes.
// Used by the check command.
func RunValidation(ctx *context.Context) error {
	return runPipes(ctx, pipe.ValidationPipes)
}

// RunExecution executes only the execution pipes.
// Should be called after RunValidation succeeds.
func RunExecution(ctx *context.Context) error {
	return runPipes(ctx, pipe.ExecutionPipes)
}

// RunAll executes validation pipes first, then execution pipes.
// Used by build, release, and snapshot commands.
func RunAll(ctx *context.Context) error {
	if err := RunValidation(ctx); err != nil {
		return err
	}
	return RunExecution(ctx)
}

// runPipes executes a slice of pipes in sequence.
func runPipes(ctx *context.Context, pipes []Piper) error {
	for _, p := range pipes {
		ctx.Logger.Infof("Running: %s", p.String())
		start := time.Now()

		if err := p.Run(ctx); err != nil {
			if isSkip(err) {
				ctx.Logger.Infof("Skipping: %v", err)
				continue
			}
			return fmt.Errorf("%s: %w", p.String(), err)
		}

		duration := time.Since(start)
		ctx.Logger.Infof("Completed: %s (%s)", p.String(), duration.Round(time.Millisecond))
	}
	return nil
}

func isSkip(err error) bool {
	var s pipe.IsSkip
	return errors.As(err, &s) && s.IsSkip()
}

// Piper is re-exported for convenience within the pipeline package.
type Piper = pipe.Piper
