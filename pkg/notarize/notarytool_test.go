package notarize

import (
	"testing"
)

func TestBuildSubmitArgs(t *testing.T) {
	tests := []struct {
		name     string
		zipPath  string
		appleID  string
		teamID   string
		password string
		want     []string
	}{
		{
			name:     "all fields populated",
			zipPath:  "/tmp/MyApp-notarize.zip",
			appleID:  "dev@example.com",
			teamID:   "TEAM123",
			password: "xxxx-xxxx-xxxx-xxxx",
			want: []string{
				"notarytool", "submit", "/tmp/MyApp-notarize.zip",
				"--apple-id", "dev@example.com",
				"--team-id", "TEAM123",
				"--password", "xxxx-xxxx-xxxx-xxxx",
				"--wait",
			},
		},
		{
			name:     "special characters in password",
			zipPath:  "/dist/App.zip",
			appleID:  "user@test.com",
			teamID:   "ABC999",
			password: "p@ss w0rd!",
			want: []string{
				"notarytool", "submit", "/dist/App.zip",
				"--apple-id", "user@test.com",
				"--team-id", "ABC999",
				"--password", "p@ss w0rd!",
				"--wait",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSubmitArgs(tt.zipPath, tt.appleID, tt.teamID, tt.password)
			if len(got) != len(tt.want) {
				t.Fatalf("BuildSubmitArgs() returned %d args, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("BuildSubmitArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildSubmitArgsAlwaysIncludesWait(t *testing.T) {
	args := BuildSubmitArgs("/path/to/app.zip", "id@example.com", "TEAM", "pass")
	last := args[len(args)-1]
	if last != "--wait" {
		t.Errorf("BuildSubmitArgs() last arg = %q, want %q", last, "--wait")
	}
}

func TestBuildSubmitArgsPasswordPosition(t *testing.T) {
	args := BuildSubmitArgs("/app.zip", "id@test.com", "T1", "secret123")
	for i, arg := range args {
		if arg == "--password" {
			if i+1 >= len(args) {
				t.Fatal("--password is the last arg with no value")
			}
			if args[i+1] != "secret123" {
				t.Errorf("password value = %q, want %q", args[i+1], "secret123")
			}
			return
		}
	}
	t.Fatal("--password flag not found in args")
}

func TestParseSubmissionID(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name: "real notarytool output",
			output: `Conducting pre-submission checks for MyApp-notarize.zip and target platform macos...
Successfully uploaded file.
  id: 2efe2717-52ef-43a5-96dc-0797e4ca1041
  path: /tmp/MyApp-notarize.zip
Waiting for processing to complete.
Current status: Accepted`,
			want: "2efe2717-52ef-43a5-96dc-0797e4ca1041",
		},
		{
			name:   "no UUID in output",
			output: "Some error output without a UUID",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
		{
			name: "invalid status with UUID",
			output: `  id: abcdef01-2345-6789-abcd-ef0123456789
  status: Invalid`,
			want: "abcdef01-2345-6789-abcd-ef0123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSubmissionID(tt.output)
			if got != tt.want {
				t.Errorf("ParseSubmissionID() = %q, want %q", got, tt.want)
			}
		})
	}
}
