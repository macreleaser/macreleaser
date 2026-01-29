package pipe

import (
	"github.com/macreleaser/macreleaser/pkg/context"
)

// Piper defines the interface for all pipeline steps.
// Each pipe represents a distinct phase in the release process and is executed
// sequentially by the pipeline. Pipes should be independent and idempotent.
type Piper interface {
	// String returns the pipe name for logging and identification.
	// This is displayed to users as the pipe executes.
	String() string

	// Run executes the pipe's logic. The context provides access to configuration,
	// logging, and cancellation signals. If the pipe completes successfully, it returns
	// nil. If the pipe encounters an error, it returns a descriptive error.
	// To indicate an intentional skip (not an error), return a SkipError via pipe.Skip().
	Run(ctx *context.Context) error
}

// IsSkip indicates that a pipe was intentionally skipped.
// This is not an error condition but a normal part of pipeline execution.
type IsSkip interface {
	IsSkip() bool
}

// SkipError represents an intentional skip of a pipeline step.
// Unlike regular errors, skips do not fail the pipeline but instead
// cause the pipeline to continue with the next pipe.
type SkipError struct {
	Reason string
}

func (e SkipError) Error() string { return e.Reason }
func (e SkipError) IsSkip() bool  { return true }

// Skip creates a new skip error with the given reason.
// Use this when a pipe determines it should not run (e.g., configuration disabled).
func Skip(reason string) SkipError {
	return SkipError{Reason: reason}
}
