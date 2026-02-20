package changelog

import (
	"strings"
	"testing"

	"github.com/macreleaser/macreleaser/pkg/config"
)

func TestGenerateFlat(t *testing.T) {
	commits := []string{"fix: resolve crash", "feat: add widget", "docs: update readme"}
	cfg := config.ChangelogConfig{}

	out, err := Generate("v1.2.0", commits, cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.HasPrefix(out, "## v1.2.0\n") {
		t.Errorf("expected heading, got:\n%s", out)
	}
	if !strings.Contains(out, "- fix: resolve crash\n") {
		t.Error("missing commit in output")
	}
	if !strings.Contains(out, "- feat: add widget\n") {
		t.Error("missing commit in output")
	}
}

func TestGenerateGrouped(t *testing.T) {
	commits := []string{"feat: add widget", "fix: resolve crash", "chore: cleanup"}
	cfg := config.ChangelogConfig{
		Groups: []config.ChangelogGroupConfig{
			{Title: "Features", Regexp: "^feat:", Order: 0},
			{Title: "Bug Fixes", Regexp: "^fix:", Order: 1},
			{Title: "Other", Order: 2}, // catch-all
		},
	}

	out, err := Generate("v1.2.0", commits, cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(out, "### Features\n") {
		t.Error("missing Features group")
	}
	if !strings.Contains(out, "### Bug Fixes\n") {
		t.Error("missing Bug Fixes group")
	}
	if !strings.Contains(out, "### Other\n") {
		t.Error("missing Other group")
	}

	// Verify ordering: Features before Bug Fixes before Other
	featIdx := strings.Index(out, "### Features")
	fixIdx := strings.Index(out, "### Bug Fixes")
	otherIdx := strings.Index(out, "### Other")
	if featIdx > fixIdx || fixIdx > otherIdx {
		t.Errorf("groups not in expected order: feat=%d, fix=%d, other=%d", featIdx, fixIdx, otherIdx)
	}
}

func TestGenerateExcludeFilter(t *testing.T) {
	commits := []string{"feat: add widget", "docs: update readme", "chore: cleanup", "fix: bug"}
	cfg := config.ChangelogConfig{
		Filters: config.ChangelogFiltersConfig{
			Exclude: []string{"^docs:", "^chore:"},
		},
	}

	out, err := Generate("v1.0.0", commits, cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if strings.Contains(out, "docs:") {
		t.Error("docs commit should be excluded")
	}
	if strings.Contains(out, "chore:") {
		t.Error("chore commit should be excluded")
	}
	if !strings.Contains(out, "feat: add widget") {
		t.Error("feat commit should be included")
	}
	if !strings.Contains(out, "fix: bug") {
		t.Error("fix commit should be included")
	}
}

func TestGenerateIncludeFilter(t *testing.T) {
	commits := []string{"feat: add widget", "docs: update readme", "fix: bug"}
	cfg := config.ChangelogConfig{
		Filters: config.ChangelogFiltersConfig{
			Include: []string{"^feat:", "^fix:"},
		},
	}

	out, err := Generate("v1.0.0", commits, cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if strings.Contains(out, "docs:") {
		t.Error("docs commit should not be included")
	}
	if !strings.Contains(out, "feat: add widget") {
		t.Error("feat commit should be included")
	}
	if !strings.Contains(out, "fix: bug") {
		t.Error("fix commit should be included")
	}
}

func TestGenerateSortAsc(t *testing.T) {
	commits := []string{"third", "second", "first"} // git order: newest first
	cfg := config.ChangelogConfig{
		Sort: "asc",
	}

	out, err := Generate("v1.0.0", commits, cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	firstIdx := strings.Index(out, "- first")
	secondIdx := strings.Index(out, "- second")
	thirdIdx := strings.Index(out, "- third")
	if firstIdx > secondIdx || secondIdx > thirdIdx {
		t.Errorf("asc sort order wrong: first=%d, second=%d, third=%d", firstIdx, secondIdx, thirdIdx)
	}
}

func TestGenerateSortDesc(t *testing.T) {
	commits := []string{"third", "second", "first"} // git order: newest first
	cfg := config.ChangelogConfig{
		Sort: "desc",
	}

	out, err := Generate("v1.0.0", commits, cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// desc keeps git order (newest first)
	thirdIdx := strings.Index(out, "- third")
	secondIdx := strings.Index(out, "- second")
	firstIdx := strings.Index(out, "- first")
	if thirdIdx > secondIdx || secondIdx > firstIdx {
		t.Errorf("desc sort order wrong: third=%d, second=%d, first=%d", thirdIdx, secondIdx, firstIdx)
	}
}

func TestGenerateEmptyCommits(t *testing.T) {
	cfg := config.ChangelogConfig{}

	out, err := Generate("v1.0.0", nil, cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.HasPrefix(out, "## v1.0.0\n") {
		t.Errorf("expected heading even with empty commits, got:\n%s", out)
	}
}

func TestGenerateCatchAllGroup(t *testing.T) {
	commits := []string{"feat: new thing", "random commit", "another unmatched"}
	cfg := config.ChangelogConfig{
		Groups: []config.ChangelogGroupConfig{
			{Title: "Features", Regexp: "^feat:", Order: 0},
			{Title: "Other Changes", Order: 1}, // catch-all
		},
	}

	out, err := Generate("v1.0.0", commits, cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(out, "### Features\n") {
		t.Error("missing Features group")
	}
	if !strings.Contains(out, "### Other Changes\n") {
		t.Error("missing catch-all group")
	}
	if !strings.Contains(out, "- random commit") {
		t.Error("unmatched commit should be in catch-all")
	}
	if !strings.Contains(out, "- another unmatched") {
		t.Error("unmatched commit should be in catch-all")
	}
}

func TestGenerateInvalidRegex(t *testing.T) {
	commits := []string{"feat: something"}
	cfg := config.ChangelogConfig{
		Filters: config.ChangelogFiltersConfig{
			Exclude: []string{"[invalid"},
		},
	}

	_, err := Generate("v1.0.0", commits, cfg)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
	if !strings.Contains(err.Error(), "invalid exclude filter") {
		t.Errorf("error = %v, want error about invalid exclude filter", err)
	}
}

func TestGenerateInvalidGroupRegex(t *testing.T) {
	commits := []string{"feat: something"}
	cfg := config.ChangelogConfig{
		Groups: []config.ChangelogGroupConfig{
			{Title: "Bad", Regexp: "[invalid", Order: 0},
		},
	}

	_, err := Generate("v1.0.0", commits, cfg)
	if err == nil {
		t.Fatal("expected error for invalid group regex")
	}
	if !strings.Contains(err.Error(), "invalid group regexp") {
		t.Errorf("error = %v, want error about invalid group regexp", err)
	}
}

func TestGenerateInvalidIncludeRegex(t *testing.T) {
	commits := []string{"feat: something"}
	cfg := config.ChangelogConfig{
		Filters: config.ChangelogFiltersConfig{
			Include: []string{"[invalid"},
		},
	}

	_, err := Generate("v1.0.0", commits, cfg)
	if err == nil {
		t.Fatal("expected error for invalid include regex")
	}
	if !strings.Contains(err.Error(), "invalid include filter") {
		t.Errorf("error = %v, want error about invalid include filter", err)
	}
}
