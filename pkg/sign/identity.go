package sign

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// identityPattern matches lines from `security find-identity -v -p codesigning` output.
// Format: "  N) <hex hash> "<identity string>""
var identityPattern = regexp.MustCompile(`^\s*\d+\)\s+[0-9A-Fa-f]+\s+"(.+)"`)

// ParseIdentityOutput parses the output of `security find-identity -v -p codesigning`
// and returns the list of identity strings (the quoted names).
func ParseIdentityOutput(output string) []string {
	var identities []string

	for _, line := range strings.Split(output, "\n") {
		matches := identityPattern.FindStringSubmatch(line)
		if len(matches) == 2 {
			identities = append(identities, matches[1])
		}
	}

	return identities
}

// ValidateIdentity checks whether configuredIdentity appears in the list of
// available identities. Returns nil on match, or an error listing available
// identities if not found.
func ValidateIdentity(configuredIdentity string, availableIdentities []string) error {
	for _, id := range availableIdentities {
		if id == configuredIdentity {
			return nil
		}
	}

	if len(availableIdentities) == 0 {
		return fmt.Errorf(
			"signing identity %q not found in keychain — no valid signing identities are installed\n"+
				"run: security find-identity -v -p codesigning",
			configuredIdentity,
		)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "signing identity %q not found in keychain\navailable identities:\n", configuredIdentity)
	for _, id := range availableIdentities {
		fmt.Fprintf(&b, "  - %s\n", id)
	}
	b.WriteString("run: security find-identity -v -p codesigning")

	return fmt.Errorf("%s", b.String())
}

// CheckIdentityInKeychain runs `security find-identity -v -p codesigning`,
// parses the output, and validates that the configured identity is present.
func CheckIdentityInKeychain(configuredIdentity string) error {
	if _, err := exec.LookPath("security"); err != nil {
		return fmt.Errorf("security command not found — this tool requires macOS")
	}

	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		return fmt.Errorf("failed to list signing identities: %s: %w", output, err)
	}

	identities := ParseIdentityOutput(output)
	return ValidateIdentity(configuredIdentity, identities)
}
