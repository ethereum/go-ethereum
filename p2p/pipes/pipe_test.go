package pipes_test

import (
	"io"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/pipes"
)

func TestTCPPipe(t *testing.T) {
	conn1, conn2, err := pipes.TCPPipe()
	if err != nil {
		t.Fatalf("Failed to create TCPPipe: %v", err)
	}
	defer conn1.Close()
	defer conn2.Close()

	// Set deadlines to prevent hanging tests.
	conn1.SetDeadline(time.Now().Add(time.Second))
	conn2.SetDeadline(time.Now().Add(time.Second))

	testMessage := "Hello!"

	// Write from one connection and read from the other
	go func() {
		if _, err := conn1.Write([]byte(testMessage)); err != nil {
			t.Errorf("Failed to write to conn1: %v", err)
		}
	}()

	buf := make([]byte, len(testMessage))
	if _, err := io.ReadFull(conn2, buf); err != nil {
		t.Fatalf("Failed to read from conn2: %v", err)
	}

	if string(buf) != testMessage {
		t.Errorf("Data mismatch: got %q, want %q", buf, testMessage)
	}
}
