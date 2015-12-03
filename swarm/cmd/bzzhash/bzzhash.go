// bzzhash
package main

import (
	"fmt"
	"io"
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
	sr := io.NewSectionReader(f, 0, stat.Size())
	chunker := storage.NewTreeChunker(storage.NewChunkerParams())
	hash := make([]byte, chunker.KeySize())
	errC := chunker.Split(hash, sr, nil, nil)
	err, ok := <-errC
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	if !ok {
		fmt.Printf("%064x\n", hash)
	}
}
