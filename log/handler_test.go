package log

import (
	"io/ioutil"
	"os"
	"testing"
)

func newTempFileWithData(data []byte) string {
	for {
		f, err := ioutil.TempFile("", "ethereum.log")
		if err != nil {
			continue
		}

		err = f.Truncate(0)
		if err != nil {
			continue
		}

		if len(data) != 0 {
			_, err = f.Write(data)
			if err != nil {
				continue
			}
		}

		name := f.Name()
		_ = f.Close()

		return name
	}
}

func TestPrepFileWithEmptyFile(t *testing.T) {
	tmpfile := newTempFileWithData(nil)

	w, err := prepFile(tmpfile)
	if err != nil {
		t.Errorf("Expect nil, got %v", err)
	}

	w.Close()

	defer os.Remove(tmpfile)
}

func TestPrepFileWithoutNewLine(t *testing.T) {
	dataWithoutNewLine := make([]byte, 100)
	for i := range dataWithoutNewLine {
		dataWithoutNewLine[i] = 'A'
	}

	tmpfile := newTempFileWithData(dataWithoutNewLine)

	w, err := prepFile(tmpfile)
	if err != nil {
		t.Errorf("Expect nil, got %v", err)
	}

	w.Close()

	defer os.Remove(tmpfile)
}

func TestPrepFileWithNewLine(t *testing.T) {
	dataWithoutNewLine := make([]byte, 100)
	for i := range dataWithoutNewLine {
		dataWithoutNewLine[i] = 'A'
	}
	dataWithoutNewLine[0] = '\n'

	tmpfile := newTempFileWithData(dataWithoutNewLine)

	w, err := prepFile(tmpfile)
	if err != nil {
		t.Errorf("Expect nil, got %v", err)
	}
	if w.count != 1 {
		t.Errorf("Expect 1, got %v", w.count)
	}

	w.Close()

	defer os.Remove(tmpfile)
}
