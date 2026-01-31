package notarize

import (
	"context"
	"strings"
	"testing"

	"github.com/macreleaser/macreleaser/pkg/config"
	macCtx "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/sirupsen/logrus"
)

func TestPipeString(t *testing.T) {
	p := Pipe{}
	expected := "notarizing application"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

func TestPipeNoApp(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	ctx := macCtx.NewContext(context.Background(), &config.Config{
		Notarize: config.NotarizeConfig{
			AppleID:  "dev@example.com",
			TeamID:   "TEAM123",
			Password: "xxxx-xxxx-xxxx-xxxx",
		},
	}, logger)

	// AppPath is empty by default from NewContext
	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected error for empty AppPath, got nil")
	}

	if !strings.Contains(err.Error(), "no .app found to notarize") {
		t.Errorf("Run() error = %v, want error containing %q", err, "no .app found to notarize")
	}
}
