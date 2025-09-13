package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

func IsValidDomain(domain string) bool {
	// Basic domain validation
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// Domain regex pattern
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

	// Check basic format
	if !domainRegex.MatchString(domain) {
		return false
	}

	// Additional checks
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		// Label cannot start or end with hyphen
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}

	return true
}

// isValidIfName checks whether the input string is a valid Linux interface name.
var ifNameRegexp = regexp.MustCompile("^[A-Za-z0-9_-]{1,15}$")

func IsValidIfname(prefix, ifname string) error {
	if !strings.HasPrefix(ifname, prefix) {
		return fmt.Errorf("interface name must start with prefix %q", prefix)
	}
	if len(ifname) > 15 {
		return fmt.Errorf("interface name %q is too long: got %d characters, max allowed is 15", ifname, len(ifname))
	}
	if !ifNameRegexp.MatchString(ifname) {
		return fmt.Errorf("interface name %q contains invalid characters: allowed are letters, digits, '_', '-'", ifname)
	}
	return nil
}

var allowedIfNameChars = regexp.MustCompile(`^[A-Za-z0-9._@-]+$`)

// IsValidPhyIfName validates a Linux interface name and returns a detailed error if invalid.
func IsValidPhyIfName(ifname string) error {
	if ifname == "" {
		return fmt.Errorf("interface name cannot be empty")
	}

	length := utf8.RuneCountInString(ifname)
	if length > 15 {
		return fmt.Errorf("interface name %q is too long: got %d characters, max allowed is 15", ifname, length)
	}

	if !allowedIfNameChars.MatchString(ifname) {
		return fmt.Errorf("interface name %q contains invalid characters: allowed are letters, digits, '.', '_', '-', '@'", ifname)
	}

	return nil
}

// IsSafeName validates a filename for use in shell commands and frontend display
// while allowing international characters (Chinese, Japanese, etc.).
// It protects against:
// - Path traversal attacks (../, ..\, etc.)
// - Shell escape sequences and special characters
// - Control characters and invisible Unicode
// - Frontend XSS attacks
// - Null bytes and other dangerous characters
func IsSafeName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Check for null bytes (always dangerous)
	if strings.Contains(name, "\x00") {
		return fmt.Errorf("name contains null byte")
	}

	// Check length limits
	if len(name) > 128 {
		return fmt.Errorf("name too long: got %d bytes, max allowed is 128", len(name))
	}

	// Reject reserved names on Windows (even on Linux for cross-platform safety)
	reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	upperFilename := strings.ToUpper(strings.TrimSpace(name))
	for _, reserved := range reservedNames {
		if upperFilename == reserved || strings.HasPrefix(upperFilename, reserved+".") {
			return fmt.Errorf("name uses reserved name: %q", reserved)
		}
	}

	// Check for path traversal patterns
	if strings.Contains(name, "..") {
		return fmt.Errorf("name contains path traversal sequence: '..'")
	}

	// Check for path separators (both Unix and Windows)
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("name contains path separators")
	}

	// Check for shell special characters that could cause command injection
	shellSpecialChars := "|&;`$(){}[]<>\"'*?~"
	if strings.ContainsAny(name, shellSpecialChars) {
		return fmt.Errorf("name contains shell special characters")
	}

	// Check each rune for dangerous characters
	for i, r := range name {
		// Reject control characters (except tab which might be intentional)
		if unicode.IsControl(r) && r != '\t' {
			return fmt.Errorf("name contains control character at position %d (U+%04X)", i, r)
		}

		// Reject invisible/formatting Unicode characters that could be used for spoofing
		if unicode.In(r, unicode.Cf, unicode.Cs, unicode.Co) {
			return fmt.Errorf("name contains invisible/formatting character at position %d (U+%04X)", i, r)
		}

		// Reject bidirectional override characters (used in Unicode spoofing attacks)
		if r >= 0x202A && r <= 0x202E || r >= 0x2066 && r <= 0x2069 {
			return fmt.Errorf("name contains bidirectional override character at position %d (U+%04X)", i, r)
		}

		// Reject zero-width characters (used in spoofing)
		if r == 0x200B || r == 0x200C || r == 0x200D || r == 0x2060 || r == 0xFEFF {
			return fmt.Errorf("name contains zero-width character at position %d (U+%04X)", i, r)
		}
	}

	// Check for HTML/XML tags that could cause XSS in frontend
	if strings.Contains(name, "<") || strings.Contains(name, ">") {
		return fmt.Errorf("name contains HTML/XML angle brackets")
	}

	// Check for leading/trailing whitespace or dots (can cause issues)
	trimmed := strings.TrimSpace(name)
	if trimmed != name {
		return fmt.Errorf("name has leading or trailing whitespace")
	}

	if strings.HasPrefix(name, ".") && name != "." && name != ".." {
		// Allow hidden files but warn about potential issues
		// This is not an error, just a design choice
	}

	if strings.HasSuffix(name, ".") {
		return fmt.Errorf("name ends with dot")
	}

	// Additional check: ensure the cleaned path equals the original
	cleaned := filepath.Clean(name)
	if cleaned != name {
		return fmt.Errorf("name contains path elements that would be normalized")
	}

	return nil
}
