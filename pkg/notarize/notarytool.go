package notarize

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var submissionIDRe = regexp.MustCompile(`id:\s*([0-9a-fA-F-]{36})`)

// BuildSubmitArgs returns the argument list for xcrun notarytool submit.
func BuildSubmitArgs(zipPath, appleID, teamID, password string) []string {
	return []string{
		"notarytool", "submit", zipPath,
		"--apple-id", appleID,
		"--team-id", teamID,
		"--password", password,
		"--wait",
	}
}

// RunSubmit submits the ZIP at zipPath to Apple's notary service using
// notarytool and waits for the result. Returns combined output and any error.
func RunSubmit(zipPath, appleID, teamID, password string) (string, error) {
	if _, err := exec.LookPath("xcrun"); err != nil {
		return "", fmt.Errorf("xcrun not found — install Xcode Command Line Tools with: xcode-select --install")
	}

	args := BuildSubmitArgs(zipPath, appleID, teamID, password)
	cmd := exec.Command("xcrun", args...)

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		if strings.Contains(output, "Unable to authenticate") {
			return output, fmt.Errorf("notarytool authentication failed — verify apple_id, team_id, and password (use an app-specific password from appleid.apple.com)")
		}
		if strings.Contains(output, "Invalid") || strings.Contains(output, "status: Invalid") {
			submissionID := ParseSubmissionID(output)
			hint := ""
			if submissionID != "" {
				hint = fmt.Sprintf(" — run: xcrun notarytool log %s to view details", submissionID)
			}
			return output, fmt.Errorf("Apple rejected the submission%s", hint) //nolint:staticcheck // proper noun
		}
		return output, fmt.Errorf("notarytool submit failed: %s: %w", output, err)
	}

	return output, nil
}

// ParseSubmissionID extracts the submission UUID from notarytool output.
// Returns an empty string if no UUID is found.
func ParseSubmissionID(output string) string {
	matches := submissionIDRe.FindStringSubmatch(output)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}
