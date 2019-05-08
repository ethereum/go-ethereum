package hidapi

import (
	"io"
)

import "C"

var writer io.Writer

func SetLogWriter(l io.Writer) {
	writer = l
}

//export goHidLog
func goHidLog(s *C.char) {
	if writer != nil {
		writer.Write([]byte(C.GoString(s)))
	}
}
