// bzzhash
package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/bzz"
	"io"
	"os"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: bzzhash <file name>")
		os.Exit(0)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("Error opening file " + os.Args[1])
		os.Exit(1)
	}

	stat, _ := f.Stat()
	sr := io.NewSectionReader(f, 0, stat.Size())
	hash := bzz.GetDPAhash(sr, nil)

	fmt.Printf("%064x\n", hash)
}
