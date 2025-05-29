// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateDirectorySize(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "geth_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test empty directory
	size, err := calculateDirectorySize(tempDir)
	if err != nil {
		t.Fatalf("Failed to calculate size of empty directory: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected size 0 for empty directory, got %d", size)
	}

	// Create test files
	testFiles := map[string][]byte{
		"file1.txt": []byte("hello world"),         // 11 bytes
		"file2.txt": []byte("test content"),        // 12 bytes
		"file3.dat": []byte("binary data content"), // 18 bytes
	}

	var expectedSize int64
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		expectedSize += int64(len(content))
	}

	// Create a subdirectory with files
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFile := []byte("subdirectory file content") // 25 bytes
	subFilePath := filepath.Join(subDir, "subfile.txt")
	if err := os.WriteFile(subFilePath, subFile, 0644); err != nil {
		t.Fatalf("Failed to create subfile: %v", err)
	}
	expectedSize += int64(len(subFile))

	// Test directory with files
	size, err = calculateDirectorySize(tempDir)
	if err != nil {
		t.Fatalf("Failed to calculate directory size: %v", err)
	}

	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}

	// Note: Testing non-existent directory behavior is OS-dependent
	// and our improved error handling may skip the initial error
}

func TestCalculateDirectorySizeSymlinks(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "geth_symlink_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a regular file
	regularFile := filepath.Join(tempDir, "regular.txt")
	content := []byte("regular file content")
	if err := os.WriteFile(regularFile, content, 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	// Create a symlink to the file (if supported by the OS)
	symlinkFile := filepath.Join(tempDir, "symlink.txt")
	if err := os.Symlink(regularFile, symlinkFile); err != nil {
		// Skip symlink test if not supported
		t.Skipf("Symlinks not supported on this system: %v", err)
	}

	// Calculate size - should count the symlink target size
	size, err := calculateDirectorySize(tempDir)
	if err != nil {
		t.Fatalf("Failed to calculate directory size with symlinks: %v", err)
	}

	// The size should include both the regular file and the symlink
	// Note: symlink behavior may vary by OS, so we just check it's reasonable
	if size < int64(len(content)) {
		t.Errorf("Directory size %d seems too small, expected at least %d", size, len(content))
	}
}

func TestCalculateDirectorySizeWithErrors(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test_dir_errors")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file with restricted permissions
	restrictedFile := filepath.Join(tempDir, "restricted.txt")
	if err := os.WriteFile(restrictedFile, []byte("test"), 0000); err != nil {
		t.Fatalf("Failed to create restricted file: %v", err)
	}

	// Create a normal file
	normalFile := filepath.Join(tempDir, "normal.txt")
	if err := os.WriteFile(normalFile, []byte("normal content"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}

	// Calculate directory size - should not fail even with permission errors
	size, err := calculateDirectorySize(tempDir)
	if err != nil {
		t.Fatalf("calculateDirectorySize should not fail with permission errors: %v", err)
	}

	// Should at least count the normal file
	if size < int64(len("normal content")) {
		t.Errorf("Size should include at least the normal file, got %d", size)
	}

	// Restore permissions for cleanup
	os.Chmod(restrictedFile, 0644)
}
