package env

import (
	"fmt"
	"os"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
)

func TestSubstituteEnvVarsNode(t *testing.T) {
	// Set up test environment variables
	if err := os.Setenv("TEST_VAR", "test-value"); err != nil {
		t.Fatalf("Failed to set TEST_VAR: %v", err)
	}
	if err := os.Setenv("VAR1", "value1"); err != nil {
		t.Fatalf("Failed to set VAR1: %v", err)
	}
	if err := os.Setenv("VAR2", "value2"); err != nil {
		t.Fatalf("Failed to set VAR2: %v", err)
	}
	if err := os.Setenv("SPECIAL_VAR", "test@#$%"); err != nil {
		t.Fatalf("Failed to set SPECIAL_VAR: %v", err)
	}
	if err := os.Setenv("EMPTY_VAR", ""); err != nil {
		t.Fatalf("Failed to set EMPTY_VAR: %v", err)
	}
	if err := os.Setenv("DANGEROUS_VAR", "line\nfeed"); err != nil {
		t.Fatalf("Failed to set DANGEROUS_VAR: %v", err)
	}
	if err := os.Setenv("CONTROL_VAR", "bad\x01value"); err != nil {
		t.Fatalf("Failed to set CONTROL_VAR: %v", err)
	}
	if err := os.Setenv("KEY_VAR", "key-value"); err != nil {
		t.Fatalf("Failed to set KEY_VAR: %v", err)
	}

	// Clean up after tests
	defer func() {
		_ = os.Unsetenv("TEST_VAR")
		_ = os.Unsetenv("VAR1")
		_ = os.Unsetenv("VAR2")
		_ = os.Unsetenv("SPECIAL_VAR")
		_ = os.Unsetenv("EMPTY_VAR")
		_ = os.Unsetenv("DANGEROUS_VAR")
		_ = os.Unsetenv("CONTROL_VAR")
		_ = os.Unsetenv("KEY_VAR")
	}()

	tests := []struct {
		name       string
		yamlInput  string
		expectErr  bool
		checkKey   string
		checkValue string
		isSequence bool
	}{
		{
			name:       "single env var substitution",
			yamlInput:  "value: env(TEST_VAR)\n",
			checkKey:   "value",
			checkValue: "test-value",
		},
		{
			name:       "env var with prefix and suffix",
			yamlInput:  "value: prefix-env(TEST_VAR)-suffix\n",
			checkKey:   "value",
			checkValue: "prefix-test-value-suffix",
		},
		{
			name:       "multiple env vars",
			yamlInput:  "value: env(VAR1)-env(VAR2)\n",
			checkKey:   "value",
			checkValue: "value1-value2",
		},
		{
			name:       "non-existent env var left as literal",
			yamlInput:  "value: env(NONEXISTENT_RANDOM_VAR_12345)\n",
			checkKey:   "value",
			checkValue: "env(NONEXISTENT_RANDOM_VAR_12345)",
		},
		{
			name:       "no env vars",
			yamlInput:  "value: no-env-here\n",
			checkKey:   "value",
			checkValue: "no-env-here",
		},
		{
			name:       "empty env var",
			yamlInput:  "value: env(EMPTY_VAR)\n",
			checkKey:   "value",
			checkValue: "",
		},
		{
			name:       "malformed env pattern",
			yamlInput:  "value: env(TEST_VAR\n",
			checkKey:   "value",
			checkValue: "env(TEST_VAR",
		},
		{
			name:       "env var with special characters",
			yamlInput:  "value: env(SPECIAL_VAR)\n",
			checkKey:   "value",
			checkValue: "test@#$%",
		},
		{
			name:       "env var with dangerous characters",
			yamlInput:  "value: env(DANGEROUS_VAR)\n",
			checkKey:   "value",
			checkValue: "line\nfeed",
		},
		{
			name:      "env var with control characters",
			yamlInput: "value: env(CONTROL_VAR)\n",
			expectErr: true,
			checkKey:  "value",
		},
		{
			name:       "mapping key remains unchanged",
			yamlInput:  "env(KEY_VAR): env(KEY_VAR)\n",
			checkKey:   "env(KEY_VAR)",
			checkValue: "key-value",
		},
		{
			name:       "sequence values substituted",
			yamlInput:  "sequence:\n  - env(VAR1)\n",
			checkKey:   "sequence",
			checkValue: "value1",
			isSequence: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := substituteAndDecode(tt.yamlInput)
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Unexpected error: %v", err)
				}
				return
			}
			if tt.expectErr {
				t.Fatalf("Expected error but got none")
			}
			if tt.isSequence {
				sequence, ok := data[tt.checkKey]
				if !ok {
					t.Fatalf("Expected key %q", tt.checkKey)
				}
				slice, ok := sequence.([]interface{})
				if !ok {
					t.Fatalf("Expected sequence to be a list")
				}
				if len(slice) != 1 {
					t.Fatalf("Expected sequence length 1, got %d", len(slice))
				}
				value, ok := slice[0].(string)
				if !ok {
					t.Fatalf("Expected sequence value to be a string")
				}
				if value != tt.checkValue {
					t.Errorf("Expected %q, got %q", tt.checkValue, value)
				}
				return
			}
			value, ok := data[tt.checkKey]
			if !ok {
				t.Fatalf("Expected key %q", tt.checkKey)
			}
			valueStr, ok := value.(string)
			if !ok {
				t.Fatalf("Expected value for key %q to be a string", tt.checkKey)
			}
			if valueStr != tt.checkValue {
				t.Errorf("Expected %q, got %q", tt.checkValue, valueStr)
			}
		})
	}
}

func TestCheckResolved(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		field   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "resolved value",
			value:   "some-actual-value",
			field:   "test.field",
			wantErr: false,
		},
		{
			name:    "empty value",
			value:   "",
			field:   "test.field",
			wantErr: false,
		},
		{
			name:    "unresolved env var",
			value:   "env(MISSING_VAR)",
			field:   "notarize.password",
			wantErr: true,
			errMsg:  "notarize.password: environment variable MISSING_VAR is not set",
		},
		{
			name:    "unresolved env var with prefix",
			value:   "prefix-env(MISSING_VAR)-suffix",
			field:   "homebrew.tap.token",
			wantErr: true,
			errMsg:  "homebrew.tap.token: environment variable MISSING_VAR is not set",
		},
		{
			name:    "malformed pattern not matched",
			value:   "env(MISSING_VAR",
			field:   "test.field",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckResolved(tt.value, tt.field)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func substituteAndDecode(input string) (map[string]any, error) {
	file, err := parser.ParseBytes([]byte(input), 0)
	if err != nil {
		return nil, err
	}
	if len(file.Docs) == 0 || file.Docs[0].Body == nil {
		return nil, fmt.Errorf("failed to parse config: empty document")
	}
	if err := SubstituteEnvVarsNode(file.Docs[0].Body); err != nil {
		return nil, err
	}
	var config map[string]any
	if err := yaml.NodeToValue(file.Docs[0].Body, &config, yaml.Strict()); err != nil {
		return nil, err
	}
	return config, nil
}
