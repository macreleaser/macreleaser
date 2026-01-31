package homebrew

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeSHA256(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		wantHash string
	}{
		{
			name:     "known content",
			content:  []byte("hello world"),
			wantHash: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "empty file",
			content:  []byte{},
			wantHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "testfile")
			if err := os.WriteFile(path, tt.content, 0644); err != nil {
				t.Fatal(err)
			}

			got, err := ComputeSHA256(path)
			if err != nil {
				t.Fatalf("ComputeSHA256() unexpected error: %v", err)
			}
			if got != tt.wantHash {
				t.Errorf("ComputeSHA256() = %q, want %q", got, tt.wantHash)
			}
		})
	}
}

func TestComputeSHA256NonExistent(t *testing.T) {
	_, err := ComputeSHA256("/nonexistent/file/path")
	if err == nil {
		t.Fatal("ComputeSHA256() expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Errorf("ComputeSHA256() error = %q, want error containing %q", err.Error(), "failed to open file")
	}
}
