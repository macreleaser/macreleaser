package homebrew

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// ComputeSHA256 computes the SHA256 hash of the file at the given path.
// Returns the lowercase hex-encoded hash string.
func ComputeSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to compute SHA256: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
