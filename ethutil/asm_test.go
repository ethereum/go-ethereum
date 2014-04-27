package ethutil

import (
	"fmt"
	"testing"
)

func TestPreProcess(t *testing.T) {
	main, init := PreProcess(`
	init {
		// init
		if a > b {
			if { 
			}
		}
	}

	main {
		// main
		if a > b {
			if c > d {
			}
		}
	}
	`)

	fmt.Println("main")
	fmt.Println(main)
	fmt.Println("init")
	fmt.Println(init)
}
