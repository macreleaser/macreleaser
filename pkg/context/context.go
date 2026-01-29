package context

import (
	"context"

	"github.com/macreleaser/macreleaser/pkg/config"
	"github.com/sirupsen/logrus"
)

// Context provides shared state for all pipes
type Context struct {
	StdCtx context.Context // Standard context for cancellation support
	Config *config.Config
	Logger *logrus.Logger
}

// NewContext creates a new context with the given standard context, config, and logger.
// If stdCtx is nil, context.Background() is used.
func NewContext(stdCtx context.Context, cfg *config.Config, logger *logrus.Logger) *Context {
	if stdCtx == nil {
		stdCtx = context.Background()
	}
	return &Context{
		StdCtx: stdCtx,
		Config: cfg,
		Logger: logger,
	}
}

// Done returns the done channel from the standard context for cancellation support
func (c *Context) Done() <-chan struct{} {
	return c.StdCtx.Done()
}

// Err returns the error from the standard context
func (c *Context) Err() error {
	return c.StdCtx.Err()
}
