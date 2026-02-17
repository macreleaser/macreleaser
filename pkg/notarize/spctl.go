package notarize

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunAssess verifies the app at appPath passes Gatekeeper assessment
// using spctl --assess. Returns combined output and any error.
func RunAssess(appPath string) (string, error) {
	if _, err := exec.LookPath("spctl"); err != nil {
		return "", fmt.Errorf("spctl not found — this tool is required for Gatekeeper verification on macOS")
	}

	cmd := exec.Command("spctl", "--assess", "--type", "execute", "--verbose", appPath)

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		if strings.Contains(output, "rejected") {
			return output, fmt.Errorf("Gatekeeper rejected the app — it may not be properly signed or notarized") //nolint:staticcheck // proper noun
		}
		return output, fmt.Errorf("spctl assess failed: %s: %w", output, err)
	}

	return output, nil
}
