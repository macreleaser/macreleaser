package changelog

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/macreleaser/macreleaser/pkg/config"
	macCtx "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/sirupsen/logrus"
)

func newCheckContext(cfg config.ChangelogConfig) *macCtx.Context {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return macCtx.NewContext(context.Background(), &config.Config{
		Changelog: cfg,
	}, logger)
}

func TestCheckPipeDisabled(t *testing.T) {
	ctx := newCheckContext(config.ChangelogConfig{Disable: true})
	err := CheckPipe{}.Run(ctx)
	if err == nil {
		t.Fatal("expected skip error, got nil")
	}
	var s interface{ IsSkip() bool }
	if !errors.As(err, &s) || !s.IsSkip() {
		t.Errorf("expected skip error, got %T: %v", err, err)
	}
}

func TestCheckPipeValidConfig(t *testing.T) {
	ctx := newCheckContext(config.ChangelogConfig{
		Sort: "desc",
		Filters: config.ChangelogFiltersConfig{
			Exclude: []string{"^docs:", "^chore:"},
		},
		Groups: []config.ChangelogGroupConfig{
			{Title: "Features", Regexp: "^feat:", Order: 0},
			{Title: "Bug Fixes", Regexp: "^fix:", Order: 1},
		},
	})

	if err := (CheckPipe{}).Run(ctx); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
}

func TestCheckPipeEmptyConfig(t *testing.T) {
	ctx := newCheckContext(config.ChangelogConfig{})
	if err := (CheckPipe{}).Run(ctx); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
}

func TestCheckPipeInvalidSort(t *testing.T) {
	ctx := newCheckContext(config.ChangelogConfig{Sort: "alphabetical"})
	err := CheckPipe{}.Run(ctx)
	if err == nil {
		t.Fatal("expected error for invalid sort")
	}
	if !strings.Contains(err.Error(), "must be") {
		t.Errorf("error = %v, want error about sort values", err)
	}
}

func TestCheckPipeInvalidExcludeRegex(t *testing.T) {
	ctx := newCheckContext(config.ChangelogConfig{
		Filters: config.ChangelogFiltersConfig{
			Exclude: []string{"[invalid"},
		},
	})
	err := CheckPipe{}.Run(ctx)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
	if !strings.Contains(err.Error(), "changelog.filters.exclude") {
		t.Errorf("error = %v, want error about exclude filter", err)
	}
}

func TestCheckPipeInvalidIncludeRegex(t *testing.T) {
	ctx := newCheckContext(config.ChangelogConfig{
		Filters: config.ChangelogFiltersConfig{
			Include: []string{"(unclosed"},
		},
	})
	err := CheckPipe{}.Run(ctx)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
	if !strings.Contains(err.Error(), "changelog.filters.include") {
		t.Errorf("error = %v, want error about include filter", err)
	}
}

func TestCheckPipeGroupMissingTitle(t *testing.T) {
	ctx := newCheckContext(config.ChangelogConfig{
		Groups: []config.ChangelogGroupConfig{
			{Title: "", Regexp: "^feat:"},
		},
	})
	err := CheckPipe{}.Run(ctx)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Errorf("error = %v, want error about missing title", err)
	}
}

func TestCheckPipeGroupInvalidRegexp(t *testing.T) {
	ctx := newCheckContext(config.ChangelogConfig{
		Groups: []config.ChangelogGroupConfig{
			{Title: "Bad", Regexp: "[invalid"},
		},
	})
	err := CheckPipe{}.Run(ctx)
	if err == nil {
		t.Fatal("expected error for invalid group regexp")
	}
	if !strings.Contains(err.Error(), "invalid regexp") {
		t.Errorf("error = %v, want error about invalid regexp", err)
	}
}

func TestCheckPipeString(t *testing.T) {
	p := CheckPipe{}
	expected := "validating changelog configuration"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}
