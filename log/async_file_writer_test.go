package log

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestWriter(t *testing.T) {
	w, err := NewAsyncFileWriter("./hello.log", 100)
	if err != nil {
		t.Fatalf(err.Error())
	}

	w.Start()
	w.Write([]byte("hello\n"))
	w.Write([]byte("world\n"))
	w.Stop()
	files, _ := ioutil.ReadDir("./")
	for _, f := range files {
		fn := f.Name()
		if strings.HasPrefix(fn, "hello") {
			t.Log(fn)
			content, _ := ioutil.ReadFile(fn)
			t.Log(content)
			os.Remove(fn)
		}
	}
}
