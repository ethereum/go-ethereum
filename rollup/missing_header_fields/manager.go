package missing_header_fields

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

const timeoutDownload = 30 * time.Minute

// Manager is responsible for managing the missing header fields file.
// It lazily downloads the file if it doesn't exist, verifies its expectedChecksum and provides the missing header fields.
type Manager struct {
	ctx              context.Context
	filePath         string
	downloadURL      string
	expectedChecksum common.Hash

	reader *Reader
}

func NewManager(ctx context.Context, filePath string, downloadURL string, expectedChecksum common.Hash) *Manager {
	return &Manager{
		ctx:              ctx,
		filePath:         filePath,
		downloadURL:      downloadURL,
		expectedChecksum: expectedChecksum,
	}
}

func (m *Manager) GetMissingHeaderFields(headerNum uint64) (difficulty uint64, stateRoot common.Hash, coinbase common.Address, nonce types.BlockNonce, extraData []byte, err error) {
	// lazy initialization: if the reader is not initialized this is the first time we read from the file
	if m.reader == nil {
		if err = m.initialize(); err != nil {
			return 0, common.Hash{}, common.Address{}, types.BlockNonce{}, nil, fmt.Errorf("failed to initialize missing header reader: %v", err)
		}
	}

	return m.reader.Read(headerNum)
}

func (m *Manager) initialize() error {
	// if the file doesn't exist, download it
	if _, err := os.Stat(m.filePath); errors.Is(err, os.ErrNotExist) {
		if err = m.downloadFile(); err != nil {
			return fmt.Errorf("failed to download file: %v", err)
		}
	}

	// verify the expectedChecksum
	f, err := os.Open(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}

	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("failed to close file: %v", err)
	}
	computedChecksum := h.Sum(nil)
	if !bytes.Equal(computedChecksum, m.expectedChecksum[:]) {
		return fmt.Errorf("expectedChecksum mismatch, expected %x, got %x. Please delete %s to restart file download", m.expectedChecksum, computedChecksum, m.filePath)
	}

	// finally initialize the reader
	reader, err := NewReader(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to create reader: %v", err)
	}

	m.reader = reader
	return nil
}

func (m *Manager) Close() error {
	if m.reader != nil {
		return m.reader.Close()
	}
	return nil
}

func (m *Manager) downloadFile() error {
	log.Info("Downloading missing header fields. This might take a while...", "url", m.downloadURL)

	downloadCtx, downloadCtxCancel := context.WithTimeout(m.ctx, timeoutDownload)
	defer downloadCtxCancel()

	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, m.downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status code %d", resp.StatusCode)
	}

	// create a temporary file
	tmpFilePath := m.filePath + ".tmp" // append .tmp to the file path
	tmpFile, err := os.Create(tmpFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	var ok bool
	defer func() {
		if !ok {
			_ = os.Remove(tmpFilePath)
		}
	}()

	// copy the response body to the temporary file and print progress
	writeCounter := NewWriteCounter(m.ctx, uint64(resp.ContentLength))
	if _, err = io.Copy(tmpFile, io.TeeReader(resp.Body, writeCounter)); err != nil {
		return fmt.Errorf("failed to copy response body: %v", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %v", err)
	}

	// rename the temporary file to the final file path
	if err = os.Rename(tmpFilePath, m.filePath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %v", err)
	}

	ok = true
	return nil
}

type WriteCounter struct {
	ctx                 context.Context
	total               uint64
	written             uint64
	lastProgressPrinted time.Time
}

func NewWriteCounter(ctx context.Context, total uint64) *WriteCounter {
	return &WriteCounter{
		ctx:   ctx,
		total: total,
	}
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.written += uint64(n)

	// check if the context is done and return early
	select {
	case <-wc.ctx.Done():
		return n, wc.ctx.Err()
	default:
	}

	wc.printProgress()

	return n, nil
}

func (wc *WriteCounter) printProgress() {
	if time.Since(wc.lastProgressPrinted) < 5*time.Second {
		return
	}
	wc.lastProgressPrinted = time.Now()

	log.Info(fmt.Sprintf("Downloading missing header fields... %d MB / %d MB", toMB(wc.written), toMB(wc.total)))
}

func toMB(bytes uint64) uint64 {
	return bytes / 1024 / 1024
}
