package sign

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunCodesign signs the app bundle at appPath with the given identity
// using --deep --force flags. When hardenedRuntime is true, --options runtime
// is included (required for notarization). Returns combined output and any error.
func RunCodesign(identity, appPath string, hardenedRuntime bool) (string, error) {
	if _, err := exec.LookPath("codesign"); err != nil {
		return "", fmt.Errorf("codesign not found — install Xcode Command Line Tools with: xcode-select --install")
	}

	args := []string{"--deep", "--force"}
	if hardenedRuntime {
		args = append(args, "--options", "runtime")
	}
	args = append(args, "--sign", identity, appPath)
	cmd := exec.Command("codesign", args...)

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		if strings.Contains(output, "resource fork, Finder information, or similar detritus") {
			return output, fmt.Errorf("codesign failed due to extended attributes — remove them with: xattr -cr %s", appPath)
		}
		return output, fmt.Errorf("codesign failed: %s: %w", output, err)
	}

	return output, nil
}

// RunVerify verifies the code signature of the app bundle at appPath
// using --deep --strict flags. Returns combined output and any error.
func RunVerify(appPath string) (string, error) {
	if _, err := exec.LookPath("codesign"); err != nil {
		return "", fmt.Errorf("codesign not found — install Xcode Command Line Tools with: xcode-select --install")
	}

	cmd := exec.Command("codesign", "--verify", "--deep", "--strict", appPath)

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		return output, fmt.Errorf("signature verification failed for %s: %s: %w", appPath, output, err)
	}

	return output, nil
}
