package archive

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
	expected := "packaging archives"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

func TestPipeNoApp(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cfg := &config.Config{
		Archive: config.ArchiveConfig{
			Formats: []string{"zip"},
		},
	}
	c := macCtx.NewContext(context.Background(), cfg, logger)
	// AppPath is empty — should fail

	err := Pipe{}.Run(c)
	if err == nil {
		t.Fatal("expected error when AppPath is empty")
	}
	if !strings.Contains(err.Error(), "no .app found to package") {
		t.Errorf("error = %v, want containing 'no .app found to package'", err)
	}
}
