package sign

import (
	"strings"
	"testing"
)

func TestParseIdentityOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name: "real format with two identities",
			output: `  1) AABBCCDDEE1234567890AABBCCDDEE12345678 "Developer ID Application: John Doe (TEAM123)"
  2) EEFF00112233445566778899AABBCCDDEEFF00 "Apple Development: john@example.com (PERSONAL)"
     2 valid identities found`,
			want: []string{
				"Developer ID Application: John Doe (TEAM123)",
				"Apple Development: john@example.com (PERSONAL)",
			},
		},
		{
			name: "single identity",
			output: `  1) AABBCCDDEE1234567890AABBCCDDEE12345678 "Developer ID Application: My Company (ABC123)"
     1 valid identities found`,
			want: []string{
				"Developer ID Application: My Company (ABC123)",
			},
		},
		{
			name:   "empty output",
			output: "",
			want:   nil,
		},
		{
			name: "no valid identities",
			output: `     0 valid identities found`,
			want:   nil,
		},
		{
			name: "malformed lines ignored",
			output: `some random text
not a valid identity line
  1) AABBCCDDEE1234567890AABBCCDDEE12345678 "Developer ID Application: Valid (TEAM)"
another bad line`,
			want: []string{
				"Developer ID Application: Valid (TEAM)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseIdentityOutput(tt.output)

			if len(got) != len(tt.want) {
				t.Fatalf("ParseIdentityOutput() returned %d identities, want %d\ngot:  %v\nwant: %v", len(got), len(tt.want), got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("identity[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidateIdentity(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		available  []string
		wantErr    bool
		errContain string
	}{
		{
			name:       "identity found",
			configured: "Developer ID Application: John Doe (TEAM123)",
			available: []string{
				"Developer ID Application: John Doe (TEAM123)",
				"Apple Development: john@example.com (PERSONAL)",
			},
			wantErr: false,
		},
		{
			name:       "identity not found with alternatives",
			configured: "Developer ID Application: Wrong Name (WRONG)",
			available: []string{
				"Developer ID Application: John Doe (TEAM123)",
				"Apple Development: john@example.com (PERSONAL)",
			},
			wantErr:    true,
			errContain: "not found in keychain",
		},
		{
			name:       "not found lists available identities",
			configured: "Developer ID Application: Wrong Name (WRONG)",
			available: []string{
				"Developer ID Application: John Doe (TEAM123)",
			},
			wantErr:    true,
			errContain: "Developer ID Application: John Doe (TEAM123)",
		},
		{
			name:       "empty available list",
			configured: "Developer ID Application: John Doe (TEAM123)",
			available:  nil,
			wantErr:    true,
			errContain: "no valid signing identities are installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIdentity(tt.configured, tt.available)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContain != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("ValidateIdentity() error = %v, want error containing %q", err, tt.errContain)
				}
			}
		})
	}
}
