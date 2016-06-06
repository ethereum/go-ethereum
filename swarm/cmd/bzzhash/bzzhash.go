// bzzhash
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ethereum/go-ethereum/swarm/storage"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

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
	chunker := storage.NewTreeChunker(storage.NewChunkerParams())
	key, err := chunker.Split(f, stat.Size(), nil, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	} else {
		fmt.Printf("%v\n", key)
	}
}
