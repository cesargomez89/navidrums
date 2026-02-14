package storage

import (
	"fmt"
	"os"
	"strings"

	"github.com/cesargomez89/navidrums/internal/constants"
)

func Sanitize(s string) string {
	// Simple sanitize for FS
	// Replace invalid chars with nothing or underscore?
	// User spec: "Sanitize all filesystem characters: Remove: <>:"/\|?*. Trim trailing dots and spaces."

	mapped := strings.Map(func(r rune) rune {
		if strings.ContainsRune("<>:\"/\\|?*", r) {
			return -1
		}
		return r
	}, s)

	return strings.TrimRight(mapped, ". ")
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, constants.DirPermissions)
}

func MoveFile(src, dst string) error {
	// Rename first
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Fallback to copy/delete?
	// For now return error
	return fmt.Errorf("failed to move %s to %s", src, dst)
}

func CreateFile(path string) (*os.File, error) {
	return os.Create(path)
}

func WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, constants.FilePermissions)
}

func RemoveFile(path string) error {
	return os.Remove(path)
}
