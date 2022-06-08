package heimdall

import (
	"testing"
)

func TestSpanURL(t *testing.T) {
	t.Parallel()

	url, err := spanURL("http://bor0", 1)
	if err != nil {
		t.Fatal("got an error", err)
	}

	const expected = "http://bor0/bor/span/1"

	if url.String() != expected {
		t.Fatalf("expected URL %q, got %q", url.String(), expected)
	}
}

func TestStateSyncURL(t *testing.T) {
	t.Parallel()

	url, err := stateSyncURL("http://bor0", 10, 100)
	if err != nil {
		t.Fatal("got an error", err)
	}

	const expected = "http://bor0/clerk/event-record/list?from-id=10&to-time=100&limit=50"

	if url.String() != expected {
		t.Fatalf("expected URL %q, got %q", url.String(), expected)
	}
}
