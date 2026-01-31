package notarize

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunStaple staples the notarization ticket to the .app at appPath
// using xcrun stapler. Returns combined output and any error.
func RunStaple(appPath string) (string, error) {
	if _, err := exec.LookPath("xcrun"); err != nil {
		return "", fmt.Errorf("xcrun not found — install Xcode Command Line Tools with: xcode-select --install")
	}

	cmd := exec.Command("xcrun", "stapler", "staple", appPath)

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		if strings.Contains(output, "Could not find ticket") {
			return output, fmt.Errorf("stapling failed — the notarization ticket was not found; ensure notarytool submission succeeded")
		}
		return output, fmt.Errorf("stapler staple failed: %s: %w", output, err)
	}

	return output, nil
}
