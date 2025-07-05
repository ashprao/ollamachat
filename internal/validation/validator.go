package validation

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidatePositiveInt validates that a string represents a positive integer
func ValidatePositiveInt(value, fieldName string) (int, error) {
	trimmed := strings.TrimSpace(value)
	num, err := strconv.Atoi(trimmed)
	if err != nil || num <= 0 {
		return 0, fmt.Errorf("%s must be a positive number", fieldName)
	}
	return num, nil
}

// ValidateNonNegativeInt validates that a string represents a non-negative integer
func ValidateNonNegativeInt(value, fieldName string) (int, error) {
	trimmed := strings.TrimSpace(value)
	num, err := strconv.Atoi(trimmed)
	if err != nil || num < 0 {
		return 0, fmt.Errorf("%s must be a non-negative number", fieldName)
	}
	return num, nil
}

// ValidateFloat validates that a string represents a float within given bounds
func ValidateFloat(value, fieldName string, min, max float64) (float64, error) {
	trimmed := strings.TrimSpace(value)
	num, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || num < min || num > max {
		return 0, fmt.Errorf("%s must be a number between %.1f and %.1f", fieldName, min, max)
	}
	return num, nil
}

// ValidateUIValues validates common UI configuration values
func ValidateUIValues(windowWidth, windowHeight, maxMessages, fontSize, sidebarWidth int) error {
	if windowWidth <= 0 {
		return fmt.Errorf("window width must be positive")
	}
	if windowHeight <= 0 {
		return fmt.Errorf("window height must be positive")
	}
	if maxMessages < 0 {
		return fmt.Errorf("max messages must be non-negative (0 to disable context)")
	}
	if fontSize <= 0 {
		return fmt.Errorf("font size must be positive")
	}
	if sidebarWidth <= 0 {
		return fmt.Errorf("sidebar width must be positive")
	}
	return nil
}
