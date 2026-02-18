package logging

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// BulletFormatter formats log entries in goreleaser-style hierarchical bullets.
//
// Entries with an "action" field produce top-level bullets:
//
//	  * loading configuration
//
// Info-level entries without "action" produce sub-bullets:
//
//	    * building scheme "MyApp"
//
// Warn-level entries produce warning sub-bullets:
//
//	    ! some warning
//
// Error-level entries produce error bullets:
//
//	  x something failed
//
// Key-value fields (excluding "action") are appended as key=value pairs.
type BulletFormatter struct{}

func (f *BulletFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var buf bytes.Buffer

	// Determine prefix and indentation based on level and action field
	action, hasAction := entry.Data["action"]

	switch {
	case hasAction:
		// Top-level action bullet
		fmt.Fprintf(&buf, "  * %s", action)
		// If there's a message beyond the action, add key-value fields
		kvs := formatFields(entry.Data, "action")
		if kvs != "" {
			fmt.Fprintf(&buf, "%s", kvs)
		}
	case entry.Level == logrus.ErrorLevel:
		fmt.Fprintf(&buf, "  x %s", entry.Message)
		kvs := formatFields(entry.Data)
		if kvs != "" {
			fmt.Fprintf(&buf, "%s", kvs)
		}
	case entry.Level == logrus.WarnLevel:
		fmt.Fprintf(&buf, "    ! %s", entry.Message)
		kvs := formatFields(entry.Data)
		if kvs != "" {
			fmt.Fprintf(&buf, "%s", kvs)
		}
	case entry.Level == logrus.InfoLevel:
		fmt.Fprintf(&buf, "    * %s", entry.Message)
		kvs := formatFields(entry.Data)
		if kvs != "" {
			fmt.Fprintf(&buf, "%s", kvs)
		}
	default:
		// Debug level — should not reach here as debug uses TextFormatter,
		// but handle gracefully
		fmt.Fprintf(&buf, "      %s", entry.Message)
	}

	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// formatFields returns a formatted string of key=value pairs, excluding
// the specified skip keys. Returns empty string if no fields remain.
func formatFields(fields logrus.Fields, skip ...string) string {
	skipSet := make(map[string]bool, len(skip))
	for _, s := range skip {
		skipSet[s] = true
	}

	// Collect and sort keys for deterministic output
	keys := make([]string, 0, len(fields))
	for k := range fields {
		if !skipSet[k] {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, fields[k]))
	}

	return "  " + strings.Join(parts, " ")
}
