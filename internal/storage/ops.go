package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cesargomez89/navidrums/internal/constants"
)

func Sanitize(s string) string {
	// Simple sanitize for FS
	// Replace invalid chars with nothing or underscore?
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

func DeleteFolderIfEmpty(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) == 0 {
		return os.Remove(dirPath)
	}
	return nil
}

func IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func VerifyFile(path, expectedHash string) (bool, error) {
	hash, err := HashFile(path)
	if err != nil {
		return false, err
	}
	return hash == expectedHash, nil
}
