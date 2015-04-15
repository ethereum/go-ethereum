package common

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
)

func Report(extra ...interface{}) {
	fmt.Fprintln(os.Stderr, "You've encountered a sought after, hard to reproduce bug. Please report this to the developers <3 https://github.com/ethereum/go-ethereum/issues")
	fmt.Fprintln(os.Stderr, extra...)

	_, file, line, _ := runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "%v:%v\n", file, line)

	debug.PrintStack()

	fmt.Fprintln(os.Stderr, "#### BUG! PLEASE REPORT ####")
}
