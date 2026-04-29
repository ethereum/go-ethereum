// Copyright 2026 The go-ethereum Authors
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
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadFileMissingDoesNotReportStale(t *testing.T) {
	db, server := newTestChecksumDB(t, "payload")
	defer server.Close()

	dst := filepath.Join(t.TempDir(), "payload.dat")
	var downloadErr error
	output := captureStdout(t, func() {
		downloadErr = db.DownloadFile(server.URL+"/payload.dat", dst)
	})
	if downloadErr != nil {
		t.Fatal(downloadErr)
	}
	if strings.Contains(output, "is stale") {
		t.Fatalf("missing file reported as stale:\n%s", output)
	}
	if !strings.Contains(output, "downloading from "+server.URL+"/payload.dat") {
		t.Fatalf("missing download log not found:\n%s", output)
	}
}

func TestDownloadFileHashMismatchReportsStale(t *testing.T) {
	db, server := newTestChecksumDB(t, "payload")
	defer server.Close()

	dst := filepath.Join(t.TempDir(), "payload.dat")
	if err := os.WriteFile(dst, []byte("old payload"), 0644); err != nil {
		t.Fatal(err)
	}
	var downloadErr error
	output := captureStdout(t, func() {
		downloadErr = db.DownloadFile(server.URL+"/payload.dat", dst)
	})
	if downloadErr != nil {
		t.Fatal(downloadErr)
	}
	if !strings.Contains(output, dst+" is stale") {
		t.Fatalf("stale download log not found:\n%s", output)
	}
}

func newTestChecksumDB(t *testing.T, content string) (*ChecksumDB, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, content)
	}))
	sum := sha256.Sum256([]byte(content))
	db, err := ParseChecksums([]byte(fmt.Sprintf("%x  payload.dat\n", sum)))
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	return db, server
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	previous := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	os.Stdout = writer
	defer func() {
		os.Stdout = previous
	}()

	fn()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
