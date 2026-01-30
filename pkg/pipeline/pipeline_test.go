package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/macreleaser/macreleaser/pkg/config"
	macContext "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/pipe"
	"github.com/sirupsen/logrus"
)

type mockPipe struct {
	name string
	err  error
}

func (m mockPipe) String() string                     { return m.name }
func (m mockPipe) Run(ctx *macContext.Context) error { return m.err }

func newContext() *macContext.Context {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	cfg := &config.Config{}
	return macContext.NewContext(context.Background(), cfg, logger)
}

func TestRunPipesSuccess(t *testing.T) {
	pipes := []Piper{
		mockPipe{name: "step1"},
		mockPipe{name: "step2"},
	}

	err := runPipes(newContext(), pipes)
	if err != nil {
		t.Fatalf("runPipes() error = %v", err)
	}
}

func TestRunPipesError(t *testing.T) {
	pipes := []Piper{
		mockPipe{name: "step1"},
		mockPipe{name: "step2", err: errors.New("something failed")},
		mockPipe{name: "step3"},
	}

	err := runPipes(newContext(), pipes)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "step2: something failed" {
		t.Errorf("error = %q, want %q", err.Error(), "step2: something failed")
	}
}

func TestRunPipesSkip(t *testing.T) {
	pipes := []Piper{
		mockPipe{name: "step1"},
		mockPipe{name: "step2", err: pipe.Skip("not needed")},
		mockPipe{name: "step3"},
	}

	err := runPipes(newContext(), pipes)
	if err != nil {
		t.Fatalf("runPipes() error = %v, want nil (skip should not fail)", err)
	}
}

func TestRunValidation(t *testing.T) {
	// Just verify RunValidation doesn't panic when called
	// Full validation requires a real config, so we test the wiring here
	ctx := newContext()
	// This will fail because config is empty, but that's expected
	err := RunValidation(ctx)
	if err == nil {
		t.Fatal("expected error with empty config")
	}
}

func TestRunAllStopsOnValidationFailure(t *testing.T) {
	ctx := newContext()
	// With empty config, validation should fail and execution should not run
	err := RunAll(ctx)
	if err == nil {
		t.Fatal("expected error with empty config")
	}
}
