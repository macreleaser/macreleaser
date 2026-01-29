package validate

import (
	"fmt"
)

// RequiredString validates that a string field is not empty
func RequiredString(value, field string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// RequiredSlice validates that a slice has at least one element
func RequiredSlice(values []string, field string) error {
	if len(values) == 0 {
		return fmt.Errorf("%s requires at least one item", field)
	}
	return nil
}

// OneOf validates that a string is one of the allowed values
func OneOf(value string, allowed []string, field string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("invalid value for %s: %s", field, value)
}

// AllOneOf validates that all items in a slice are in the allowed set
func AllOneOf(values []string, allowed []string, field string) error {
	// Convert allowed slice to map for O(1) lookup
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = struct{}{}
	}

	for _, v := range values {
		if _, ok := allowedSet[v]; !ok {
			return fmt.Errorf("invalid %s: %s", field, v)
		}
	}
	return nil
}
