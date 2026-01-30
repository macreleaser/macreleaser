package archive

import (
	"fmt"
	"os/exec"
)

// CreateDMG creates a DMG disk image containing the given .app using hdiutil.
// volumeName is the name shown when the DMG is mounted.
// Returns on success or error.
func CreateDMG(appPath, outputPath, volumeName string) error {
	if _, err := exec.LookPath("hdiutil"); err != nil {
		return fmt.Errorf("hdiutil not found — this tool is required for DMG packaging on macOS")
	}

	cmd := exec.Command("hdiutil", "create",
		"-volname", volumeName,
		"-srcfolder", appPath,
		"-ov",
		"-format", "UDZO",
		outputPath,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create DMG image: %s: %w", string(out), err)
	}

	return nil
}
