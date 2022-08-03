package debug

import (
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
