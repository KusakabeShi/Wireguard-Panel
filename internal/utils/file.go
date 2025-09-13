package utils

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"wg-panel/internal/logging"
)

// WriteFileAtomic writes data to a file atomically by writing to a temporary file first
func WriteFileAtomic(filename string, data []byte, perm os.FileMode) error {
	logging.LogInfo("Writing file %s (%d bytes)", filename, len(data))
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory:-> %v", err)
	}

	tmpFile := filename + ".tmp"

	if err := os.WriteFile(tmpFile, data, perm); err != nil {
		return fmt.Errorf("failed to write temporary file:-> %v", err)
	}

	if err := os.Rename(tmpFile, filename); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename temporary file:-> %v", err)
	}

	logging.LogInfo("Successfully wrote file %s", filename)
	return nil
}

// GenerateRandomString generates a random string with optional prefix
func GenerateRandomString(prefix string, length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	if prefix != "" {
		return prefix + string(b), nil
	}
	return string(b), nil
}

func StringPointerEqual(str1, str2 *string, emptyIsNull bool) bool {
	if emptyIsNull {
		if str1 != nil && *str1 == "" {
			str1 = nil
		}
		if str2 != nil && *str2 == "" {
			str2 = nil
		}
	}
	if str1 == nil && str2 == nil {
		return true
	}
	if str1 == nil && str2 != nil {
		return false
	}
	if str1 != nil && str2 == nil {
		return false
	}
	return *str1 == *str2
}
