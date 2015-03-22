package common

import (
	"fmt"
	"time"
)

func Bench(pre string, cb func()) {
	start := time.Now()
	cb()
	fmt.Println(pre, ": took:", time.Since(start))
}
