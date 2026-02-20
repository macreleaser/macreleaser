package changelog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/macreleaser/macreleaser/pkg/config"
)

// Generate produces a markdown changelog from commit messages.
// The version string is used as a heading; commits are filtered, sorted,
// and optionally grouped according to cfg.
func Generate(version string, commits []string, cfg config.ChangelogConfig) (string, error) {
	filtered, err := filterCommits(commits, cfg.Filters)
	if err != nil {
		return "", err
	}

	sorted := sortEntries(filtered, cfg.Sort)

	if len(cfg.Groups) > 0 {
		return formatGrouped(version, sorted, cfg.Groups)
	}
	return formatFlat(version, sorted), nil
}

// filterCommits applies include/exclude regex filters to commits.
// If include patterns are set, only commits matching at least one are kept.
// Then any commits matching an exclude pattern are removed.
func filterCommits(commits []string, filters config.ChangelogFiltersConfig) ([]string, error) {
	result := commits

	if len(filters.Include) > 0 {
		var includeRegexps []*regexp.Regexp
		for _, pattern := range filters.Include {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid include filter %q: %w", pattern, err)
			}
			includeRegexps = append(includeRegexps, re)
		}

		var included []string
		for _, c := range result {
			for _, re := range includeRegexps {
				if re.MatchString(c) {
					included = append(included, c)
					break
				}
			}
		}
		result = included
	}

	if len(filters.Exclude) > 0 {
		var excludeRegexps []*regexp.Regexp
		for _, pattern := range filters.Exclude {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid exclude filter %q: %w", pattern, err)
			}
			excludeRegexps = append(excludeRegexps, re)
		}

		var filtered []string
		for _, c := range result {
			excluded := false
			for _, re := range excludeRegexps {
				if re.MatchString(c) {
					excluded = true
					break
				}
			}
			if !excluded {
				filtered = append(filtered, c)
			}
		}
		result = filtered
	}

	return result, nil
}

// sortEntries sorts commits. Git log returns newest-first (desc).
// "asc" reverses to oldest-first; "desc" (default) keeps git order.
func sortEntries(commits []string, sortOrder string) []string {
	sorted := make([]string, len(commits))
	copy(sorted, commits)

	if strings.EqualFold(sortOrder, "asc") {
		for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
			sorted[i], sorted[j] = sorted[j], sorted[i]
		}
	}

	return sorted
}

// formatGrouped formats commits into titled groups sorted by Order.
// A group with an empty Regexp acts as a catch-all for unmatched commits.
func formatGrouped(version string, commits []string, groups []config.ChangelogGroupConfig) (string, error) {
	// Sort groups by Order
	sortedGroups := make([]config.ChangelogGroupConfig, len(groups))
	copy(sortedGroups, groups)
	sort.Slice(sortedGroups, func(i, j int) bool {
		return sortedGroups[i].Order < sortedGroups[j].Order
	})

	// Compile regexps and bucket commits
	type groupBucket struct {
		title   string
		re      *regexp.Regexp // nil means catch-all
		commits []string
	}

	buckets := make([]groupBucket, len(sortedGroups))
	for i, g := range sortedGroups {
		buckets[i].title = g.Title
		if g.Regexp != "" {
			re, err := regexp.Compile(g.Regexp)
			if err != nil {
				return "", fmt.Errorf("invalid group regexp %q: %w", g.Regexp, err)
			}
			buckets[i].re = re
		}
	}

	// Assign each commit to the first matching group
	for _, c := range commits {
		matched := false
		for i := range buckets {
			if buckets[i].re != nil && buckets[i].re.MatchString(c) {
				buckets[i].commits = append(buckets[i].commits, c)
				matched = true
				break
			}
		}
		if !matched {
			// Assign to catch-all (first group with nil re)
			for i := range buckets {
				if buckets[i].re == nil {
					buckets[i].commits = append(buckets[i].commits, c)
					matched = true
					break
				}
			}
			// If no catch-all, commit is silently dropped
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n", version)

	for _, bucket := range buckets {
		if len(bucket.commits) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n### %s\n\n", bucket.title)
		for _, c := range bucket.commits {
			fmt.Fprintf(&b, "- %s\n", c)
		}
	}

	return b.String(), nil
}

// formatFlat formats commits as a simple bullet list under a version heading.
func formatFlat(version string, commits []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", version)

	for _, c := range commits {
		fmt.Fprintf(&b, "- %s\n", c)
	}

	return b.String()
}
