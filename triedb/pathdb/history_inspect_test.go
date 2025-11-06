package pathdb

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

// fakeAncientReader is a minimal AncientReader used for testing sanitizeRange.
// It only provides Tail and Ancients meaningful values; other methods are
// unimplemented and should not be called by these tests.
type fakeAncientReader struct {
	tail uint64
	head uint64
}

// Implement ethdb.AncientReaderOp
func (f *fakeAncientReader) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeAncientReader) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeAncientReader) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeAncientReader) Ancients() (uint64, error)               { return f.head, nil }
func (f *fakeAncientReader) Tail() (uint64, error)                   { return f.tail, nil }
func (f *fakeAncientReader) AncientSize(kind string) (uint64, error) { return 0, nil }

// Implement ethdb.AncientReader
func (f *fakeAncientReader) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
	return fn(f)
}

func TestSanitizeRange_SingleItem_AutoBounds(t *testing.T) {
	// tail=4, head=5 => only one history with id=5
	fr := &fakeAncientReader{tail: 4, head: 5}
	first, last, err := sanitizeRange(0, 0, fr)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if first != 5 || last != 5 {
		t.Fatalf("want first=5,last=5; got first=%d,last=%d", first, last)
	}
}

func TestSanitizeRange_ExplicitSingleItem(t *testing.T) {
	fr := &fakeAncientReader{tail: 10, head: 20}
	first, last, err := sanitizeRange(15, 15, fr)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if first != 15 || last != 15 {
		t.Fatalf("want first=15,last=15; got first=%d,last=%d", first, last)
	}
}

func TestSanitizeRange_EmptyStore_Error(t *testing.T) {
	// head==tail indicates no histories available
	fr := &fakeAncientReader{tail: 10, head: 10}
	_, _, err := sanitizeRange(0, 0, fr)
	if err == nil {
		t.Fatalf("expected error for empty store, got nil")
	}
}

func TestSanitizeRange_AutoClampBounds(t *testing.T) {
	fr := &fakeAncientReader{tail: 5, head: 10}
	// start below first, end above last -> should clamp to [6,10]
	first, last, err := sanitizeRange(3, 12, fr)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if first != 6 || last != 10 {
		t.Fatalf("want first=6,last=10; got first=%d,last=%d", first, last)
	}
}

func TestSanitizeRange_StartGreaterThanEndAfterClamp_Error(t *testing.T) {
	fr := &fakeAncientReader{tail: 5, head: 7}
	// start beyond last while end inside -> becomes first=9,last=7 -> error
	_, _, err := sanitizeRange(9, 6, fr)
	if err == nil {
		t.Fatalf("expected error when first > last after clamping, got nil")
	}
}

func TestSanitizeRange_LastEqualsHead(t *testing.T) {
	fr := &fakeAncientReader{tail: 50, head: 100}
	first, last, err := sanitizeRange(0, 0, fr)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if last != 100 {
		t.Fatalf("want last=head=100; got last=%d", last)
	}
	if first != 51 {
		t.Fatalf("want first=tail+1=51; got first=%d", first)
	}
}
