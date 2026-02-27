package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cesargomez89/navidrums/internal/constants"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Normal Name", "Normal Name"},
		{"Slash/Name", "SlashName"},
		{"Colon:Name", "ColonName"},
		{"Trailing Dot.", "Trailing Dot"},
		{"AC/DC", "ACDC"},
		{"<Invalid>", "Invalid"},
	}

	for _, tt := range tests {
		got := Sanitize(tt.input)
		if got != tt.expected {
			t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestHashFile(t *testing.T) {
	// Create a temp file with known content
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	content := []byte("hello world")
	err := os.WriteFile(tmpFile, content, constants.FilePermissions)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Test HashFile
	hash, err := HashFile(tmpFile)
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	// SHA256 of "hello world" is b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
	expectedHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expectedHash {
		t.Errorf("HashFile() = %s, want %s", hash, expectedHash)
	}

	// Test HashFile on non-existent file
	_, err = HashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestVerifyFile(t *testing.T) {
	// Create a temp file with known content
	tmpFile := filepath.Join(t.TempDir(), "verify_test.txt")
	content := []byte("test content for verification")
	err := os.WriteFile(tmpFile, content, constants.FilePermissions)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Get the correct hash
	correctHash, _ := HashFile(tmpFile)

	// Test VerifyFile with correct hash
	match, err := VerifyFile(tmpFile, correctHash)
	if err != nil {
		t.Fatalf("VerifyFile failed: %v", err)
	}
	if !match {
		t.Error("Expected file to match correct hash")
	}

	// Test VerifyFile with wrong hash
	match, err = VerifyFile(tmpFile, "wrong_hash")
	if err != nil {
		t.Fatalf("VerifyFile failed: %v", err)
	}
	if match {
		t.Error("Expected file to not match wrong hash")
	}

	// Test VerifyFile on non-existent file
	_, err = VerifyFile("/nonexistent/file.txt", "any_hash")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestEnsureDir(t *testing.T) {
	// Create a temp directory
	tmpBase := t.TempDir()
	newDir := filepath.Join(tmpBase, "subdir", "nested")

	// Test EnsureDir creates nested directories
	err := EnsureDir(newDir)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected path to be a directory")
	}

	// Test EnsureDir on existing directory (should not fail)
	err = EnsureDir(newDir)
	if err != nil {
		t.Errorf("EnsureDir on existing dir failed: %v", err)
	}
}

func TestDeleteFolderIfEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Test deleting empty folder
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(emptyDir, constants.DirPermissions); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	err := DeleteFolderIfEmpty(emptyDir)
	if err != nil {
		t.Fatalf("DeleteFolderIfEmpty failed: %v", err)
	}

	// Verify folder is deleted
	if _, statErr := os.Stat(emptyDir); !os.IsNotExist(statErr) {
		t.Error("Expected empty folder to be deleted")
	}

	// Test keeping non-empty folder
	nonEmptyDir := filepath.Join(tmpDir, "nonempty")
	if mErr := os.MkdirAll(nonEmptyDir, constants.DirPermissions); mErr != nil {
		t.Fatalf("MkdirAll failed: %v", mErr)
	}
	if wErr := os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("content"), constants.FilePermissions); wErr != nil {
		t.Fatalf("WriteFile failed: %v", wErr)
	}

	err = DeleteFolderIfEmpty(nonEmptyDir)
	if err != nil {
		t.Fatalf("DeleteFolderIfEmpty failed: %v", err)
	}

	// Verify folder still exists
	if _, statErr := os.Stat(nonEmptyDir); os.IsNotExist(statErr) {
		t.Error("Expected non-empty folder to NOT be deleted")
	}

	// Test on non-existent folder (should not error)
	err = DeleteFolderIfEmpty("/nonexistent/folder")
	if err != nil {
		t.Errorf("DeleteFolderIfEmpty on nonexistent should not error: %v", err)
	}
}

func TestDeleteFolderWithCover(t *testing.T) {
	tmpDir := t.TempDir()

	// Test deleting empty folder
	emptyDir := filepath.Join(tmpDir, "empty")
	if mkdirErr := os.MkdirAll(emptyDir, constants.DirPermissions); mkdirErr != nil {
		t.Fatalf("MkdirAll failed: %v", mkdirErr)
	}

	err := DeleteFolderWithCover(emptyDir)
	if err != nil {
		t.Fatalf("DeleteFolderWithCover failed: %v", err)
	}

	if _, statErr := os.Stat(emptyDir); !os.IsNotExist(statErr) {
		t.Error("Expected empty folder to be deleted")
	}

	// Test deleting folder with only cover.jpg
	coverOnlyDir := filepath.Join(tmpDir, "coveronly")
	if mkdirErr := os.MkdirAll(coverOnlyDir, constants.DirPermissions); mkdirErr != nil {
		t.Fatalf("MkdirAll failed: %v", mkdirErr)
	}
	coverPath := filepath.Join(coverOnlyDir, "cover.jpg")
	if writeErr := os.WriteFile(coverPath, []byte("fake image"), constants.FilePermissions); writeErr != nil {
		t.Fatalf("WriteFile failed: %v", writeErr)
	}

	err = DeleteFolderWithCover(coverOnlyDir)
	if err != nil {
		t.Fatalf("DeleteFolderWithCover failed: %v", err)
	}

	if _, statErr := os.Stat(coverOnlyDir); !os.IsNotExist(statErr) {
		t.Error("Expected folder with only cover.jpg to be deleted")
	}

	// Test keeping folder with cover.jpg and other files
	multiDir := filepath.Join(tmpDir, "multi")
	if mkdirErr := os.MkdirAll(multiDir, constants.DirPermissions); mkdirErr != nil {
		t.Fatalf("MkdirAll failed: %v", mkdirErr)
	}
	if writeErr := os.WriteFile(filepath.Join(multiDir, "cover.jpg"), []byte("fake image"), constants.FilePermissions); writeErr != nil {
		t.Fatalf("WriteFile failed: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(multiDir, "track.flac"), []byte("audio"), constants.FilePermissions); writeErr != nil {
		t.Fatalf("WriteFile failed: %v", writeErr)
	}

	err = DeleteFolderWithCover(multiDir)
	if err != nil {
		t.Fatalf("DeleteFolderWithCover failed: %v", err)
	}

	if _, statErr := os.Stat(multiDir); os.IsNotExist(statErr) {
		t.Error("Expected folder with multiple files to NOT be deleted")
	}

	// Test on non-existent folder (should not error)
	err = DeleteFolderWithCover("/nonexistent/folder")
	if err != nil {
		t.Errorf("DeleteFolderWithCover on nonexistent should not error: %v", err)
	}
}

func TestIsNotExist(t *testing.T) {
	// Test with existing file
	tmpFile := filepath.Join(t.TempDir(), "exists.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), constants.FilePermissions); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	err := os.Remove(tmpFile)
	if err != nil {
		t.Fatalf("Failed to remove file: %v", err)
	}

	// Test IsNotExist with removed file
	_, err = os.Stat(tmpFile)
	if !IsNotExist(err) {
		t.Error("Expected error to indicate file does not exist")
	}

	// Test IsNotExist with no error
	existingFile := filepath.Join(t.TempDir(), "still_exists.txt")
	if wErr := os.WriteFile(existingFile, []byte("test"), constants.FilePermissions); wErr != nil {
		t.Fatalf("WriteFile failed: %v", wErr)
	}

	_, err = os.Stat(existingFile)
	if IsNotExist(err) {
		t.Error("Expected no error for existing file")
	}
}

func TestMoveFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	// Create source file
	err := os.WriteFile(src, []byte("move test"), constants.FilePermissions)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Test MoveFile
	err = MoveFile(src, dst)
	if err != nil {
		t.Fatalf("MoveFile failed: %v", err)
	}

	// Verify source is gone
	if _, statErr := os.Stat(src); !os.IsNotExist(statErr) {
		t.Error("Expected source file to be removed")
	}

	// Verify destination exists
	content, err := os.ReadFile(dst) //nolint:gosec
	if err != nil {
		t.Fatalf("Failed to read destination: %v", err)
	}
	if string(content) != "move test" {
		t.Errorf("Expected content 'move test', got %s", string(content))
	}
}
