package debug

import (
	"fmt"
	"runtime"
)

// Callers returns given number of callers with packages
func Callers(show int) []string {
	fpcs := make([]uintptr, show)

	n := runtime.Callers(2, fpcs)
	if n == 0 {
		return nil
	}

	callers := make([]string, 0, len(fpcs))

	for _, p := range fpcs {
		caller := runtime.FuncForPC(p - 1)
		if caller == nil {
			continue
		}

		callers = append(callers, caller.Name())
	}

	return callers
}

func CodeLine() (string, string, int) {
	pc, filename, line, _ := runtime.Caller(1)
	return runtime.FuncForPC(pc).Name(), filename, line
}

func CodeLineStr() string {
	pc, filename, line, _ := runtime.Caller(1)
	return fmt.Sprintf("%s:%d - %s", filename, line, runtime.FuncForPC(pc).Name())
}

func Stack(all bool) []byte {
	buf := make([]byte, 4096)

	for {
		n := runtime.Stack(buf, all)
		if n < len(buf) {
			return buf[:n]
		}

		buf = make([]byte, 2*len(buf))
	}
}
