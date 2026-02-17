package env

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml/ast"
)

// envVarPattern matches env(VAR_NAME) patterns
var envVarPattern = regexp.MustCompile(`env\(([^)]+)\)`)

// disallowedControlChars contains control characters that are not safe to inject
// into configuration values. Newlines and tabs are allowed for multiline secrets.
var disallowedControlChars = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]`)

// SubstituteEnvVarsNode replaces env(VAR_NAME) patterns in YAML value nodes only.
// Map keys are not modified.
func SubstituteEnvVarsNode(node ast.Node) error {
	if node == nil {
		return nil
	}
	return substituteNode(node, true)
}

func substituteNode(node ast.Node, inValue bool) error {
	switch n := node.(type) {
	case *ast.DocumentNode:
		if n.Body == nil {
			return nil
		}
		return substituteNode(n.Body, true)
	case *ast.MappingNode:
		for _, value := range n.Values {
			if err := substituteNode(value, inValue); err != nil {
				return err
			}
		}
		return nil
	case *ast.MappingValueNode:
		if n.Value == nil {
			return nil
		}
		return substituteNode(n.Value, true)
	case *ast.SequenceNode:
		for _, value := range n.Values {
			if err := substituteNode(value, true); err != nil {
				return err
			}
		}
		return nil
	case *ast.TagNode:
		if n.Value == nil {
			return nil
		}
		return substituteNode(n.Value, inValue)
	case *ast.AnchorNode:
		if n.Value == nil {
			return nil
		}
		return substituteNode(n.Value, inValue)
	case *ast.LiteralNode:
		if !inValue || n.Value == nil {
			return nil
		}
		replaced, err := replaceEnvVarsInString(n.Value.Value)
		if err != nil {
			return err
		}
		n.Value.Value = replaced
		return nil
	case *ast.StringNode:
		if !inValue {
			return nil
		}
		replaced, err := replaceEnvVarsInString(n.Value)
		if err != nil {
			return err
		}
		n.Value = replaced
		return nil
	case *ast.MappingKeyNode, *ast.AliasNode, *ast.DirectiveNode:
		return nil
	default:
		return nil
	}
}

// CheckResolved verifies that a config value contains no unresolved env(...)
// references. Call this in CheckPipes after skip guards to produce clear errors
// like: "notarize.password: environment variable APPLE_PASSWORD is not set".
func CheckResolved(value, field string) error {
	matches := envVarPattern.FindAllStringSubmatch(value, -1)
	for _, m := range matches {
		return fmt.Errorf("%s: environment variable %s is not set", field, m[1])
	}
	return nil
}

func replaceEnvVarsInString(input string) (string, error) {
	var err error
	result := envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(match, "env("), ")")
		value, ok := os.LookupEnv(key)
		if !ok {
			// Leave unresolved — CheckResolved will catch it later
			return match
		}

		if disallowedControlChars.MatchString(value) {
			err = fmt.Errorf("environment variable %s contains disallowed control characters", key)
			return ""
		}

		return value
	})

	if err != nil {
		return "", err
	}
	return result, nil
}
