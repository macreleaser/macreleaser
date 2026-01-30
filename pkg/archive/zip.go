package archive

import (
	"fmt"
	"os/exec"
)

// CreateZip creates a ZIP archive of the given .app using ditto.
// ditto preserves macOS resource forks and extended attributes.
// Returns the output path of the created .zip file.
func CreateZip(appPath, outputPath string) error {
	if _, err := exec.LookPath("ditto"); err != nil {
		return fmt.Errorf("ditto not found — this tool is required for ZIP packaging on macOS")
	}

	cmd := exec.Command("ditto", "-c", "-k", "--sequesterRsrc", "--keepParent", appPath, outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create ZIP archive: %s: %w", string(out), err)
	}

	return nil
}
