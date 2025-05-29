// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package download

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadFileMessages(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test file content"))
	}))
	defer server.Close()

	// Create test checksum database
	testContent := "test file content"
	hash := sha256.Sum256([]byte(testContent))
	expectedHash := hex.EncodeToString(hash[:])

	checksumData := fmt.Sprintf("# %s\n%s testfile.txt\n", server.URL, expectedHash)
	csdb, err := ParseChecksums([]byte(checksumData))
	if err != nil {
		t.Fatalf("Failed to parse checksums: %v", err)
	}

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "testfile.txt")

	tests := []struct {
		name           string
		setupFile      func() error
		expectedOutput string
	}{
		{
			name:           "file not found",
			setupFile:      func() error { return nil }, // Don't create file
			expectedOutput: "not found, downloading...",
		},
		{
			name: "file exists with correct hash",
			setupFile: func() error {
				return os.WriteFile(testFile, []byte(testContent), 0644)
			},
			expectedOutput: "is up-to-date",
		},
		{
			name: "file exists with wrong hash",
			setupFile: func() error {
				return os.WriteFile(testFile, []byte("wrong content"), 0644)
			},
			expectedOutput: "is stale (hash mismatch)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up from previous test
			os.Remove(testFile)

			// Setup test file if needed
			if err := tt.setupFile(); err != nil {
				t.Fatalf("Failed to setup test file: %v", err)
			}

			// Capture output
			var buf bytes.Buffer
			originalStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the download
			url := server.URL + "/testfile.txt"
			err := csdb.DownloadFile(url, testFile)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = originalStdout
			io.Copy(&buf, r)
			output := buf.String()

			// For "up-to-date" case, we expect no error and no download
			if tt.name == "file exists with correct hash" {
				if err != nil {
					t.Errorf("Expected no error for up-to-date file, got: %v", err)
				}
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got: %q", tt.expectedOutput, output)
				}
				return
			}

			// For other cases, we expect successful download
			if err != nil {
				t.Errorf("Download failed: %v", err)
			}

			if !strings.Contains(output, tt.expectedOutput) {
				t.Errorf("Expected output to contain %q, got: %q", tt.expectedOutput, output)
			}

			// Verify file was downloaded correctly
			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Errorf("Failed to read downloaded file: %v", err)
			}
			if string(content) != testContent {
				t.Errorf("Downloaded content mismatch: got %q, want %q", string(content), testContent)
			}
		})
	}
}

func TestVerifyHashErrorTypes(t *testing.T) {
	tempDir := t.TempDir()

	// Test file not found
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	err := verifyHash(nonExistentFile, "somehash")
	if !os.IsNotExist(err) {
		t.Errorf("Expected os.IsNotExist error for non-existent file, got: %v", err)
	}

	// Test hash mismatch
	testFile := filepath.Join(tempDir, "test.txt")
	content := "test content"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	wrongHash := "wronghash"
	err = verifyHash(testFile, wrongHash)
	if err == nil {
		t.Error("Expected hash mismatch error, got nil")
	}
	if os.IsNotExist(err) {
		t.Error("Hash mismatch should not be treated as file not found")
	}

	// Test correct hash
	hash := sha256.Sum256([]byte(content))
	correctHash := hex.EncodeToString(hash[:])
	err = verifyHash(testFile, correctHash)
	if err != nil {
		t.Errorf("Expected no error for correct hash, got: %v", err)
	}
}
