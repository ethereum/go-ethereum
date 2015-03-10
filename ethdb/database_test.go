package ethdb

/*
import (
	"bytes"
	"testing"
)

func TestCompression(t *testing.T) {
	db, err := NewLDBDatabase("testdb")
	if err != nil {
		t.Fatal(err)
	}

	in := make([]byte, 10)
	db.Put([]byte("test1"), in)
	out, err := db.Get([]byte("test1"))
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(out, in) != 0 {
		t.Error("put get", in, out)
	}
}
*/
