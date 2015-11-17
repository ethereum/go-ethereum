// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/peterh/liner"
)

const (
	importBatchSize = 2500
)

var (
	interruptCallbacks = []func(os.Signal){}
)

func openLogFile(Datadir string, filename string) *os.File {
	path := common.AbsolutePath(Datadir, filename)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("error opening log file '%s': %v", filename, err))
	}
	return file
}

func PromptConfirm(prompt string) (bool, error) {
	var (
		input string
		err   error
	)
	prompt = prompt + " [y/N] "

	// if liner.TerminalSupported() {
	// 	fmt.Println("term")
	// 	lr := liner.NewLiner()
	// 	defer lr.Close()
	// 	input, err = lr.Prompt(prompt)
	// } else {
	fmt.Print(prompt)
	input, err = bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Println()
	// }

	if len(input) > 0 && strings.ToUpper(input[:1]) == "Y" {
		return true, nil
	} else {
		return false, nil
	}

	return false, err
}

func PromptPassword(prompt string, warnTerm bool) (string, error) {
	if liner.TerminalSupported() {
		lr := liner.NewLiner()
		defer lr.Close()
		return lr.PasswordPrompt(prompt)
	}
	if warnTerm {
		fmt.Println("!! Unsupported terminal, password will be echoed.")
	}
	fmt.Print(prompt)
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Println()
	return input, err
}

// Fatalf formats a message to standard error and exits the program.
// The message is also printed to standard output if standard error
// is redirected to a different file.
func Fatalf(format string, args ...interface{}) {
	w := io.MultiWriter(os.Stdout, os.Stderr)
	outf, _ := os.Stdout.Stat()
	errf, _ := os.Stderr.Stat()
	if outf != nil && errf != nil && os.SameFile(outf, errf) {
		w = os.Stderr
	}
	fmt.Fprintf(w, "Fatal: "+format+"\n", args...)
	logger.Flush()
	os.Exit(1)
}

func StartNode(stack *node.Node) {
	if err := stack.Start(); err != nil {
		Fatalf("Error starting protocol stack: %v", err)
	}
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)
		defer signal.Stop(sigc)
		<-sigc
		glog.V(logger.Info).Infoln("Got interrupt, shutting down...")
		go stack.Stop()
		logger.Flush()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				glog.V(logger.Info).Infoln("Already shutting down, please be patient.")
				glog.V(logger.Info).Infoln("Interrupt", i-1, "more times to induce panic.")
			}
		}
		glog.V(logger.Error).Infof("Force quitting: this might not end so well.")
		panic("boom")
	}()
}

func FormatTransactionData(data string) []byte {
	d := common.StringToByteFunc(data, func(s string) (ret []byte) {
		slice := regexp.MustCompile("\\n|\\s").Split(s, 1000000000)
		for _, dataItem := range slice {
			d := common.FormatData(dataItem)
			ret = append(ret, d...)
		}
		return
	})

	return d
}

func ImportChain(chain *core.BlockChain, fn string) error {
	// Watch for Ctrl-C while the import is running.
	// If a signal is received, the import will stop at the next batch.
	interrupt := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	defer close(interrupt)
	go func() {
		if _, ok := <-interrupt; ok {
			glog.Info("caught interrupt during import, will stop at next batch")
		}
		close(stop)
	}()
	checkInterrupt := func() bool {
		select {
		case <-stop:
			return true
		default:
			return false
		}
	}

	glog.Infoln("Importing blockchain", fn)
	fh, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer fh.Close()
	stream := rlp.NewStream(fh, 0)

	// Run actual the import.
	blocks := make(types.Blocks, importBatchSize)
	n := 0
	for batch := 0; ; batch++ {
		// Load a batch of RLP blocks.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		i := 0
		for ; i < importBatchSize; i++ {
			var b types.Block
			if err := stream.Decode(&b); err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("at block %d: %v", n, err)
			}
			// don't import first block
			if b.NumberU64() == 0 {
				i--
				continue
			}
			blocks[i] = &b
			n++
		}
		if i == 0 {
			break
		}
		// Import the batch.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		if hasAllBlocks(chain, blocks[:i]) {
			glog.Infof("skipping batch %d, all blocks present [%x / %x]",
				batch, blocks[0].Hash().Bytes()[:4], blocks[i-1].Hash().Bytes()[:4])
			continue
		}

		if _, err := chain.InsertChain(blocks[:i]); err != nil {
			return fmt.Errorf("invalid block %d: %v", n, err)
		}
	}
	return nil
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash()) {
			return false
		}
	}
	return true
}

func ExportChain(blockchain *core.BlockChain, fn string) error {
	glog.Infoln("Exporting blockchain to", fn)
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := blockchain.Export(fh); err != nil {
		return err
	}
	glog.Infoln("Exported blockchain to", fn)
	return nil
}

func ExportAppendChain(blockchain *core.BlockChain, fn string, first uint64, last uint64) error {
	glog.Infoln("Exporting blockchain to", fn)
	// TODO verify mode perms
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := blockchain.ExportN(fh, first, last); err != nil {
		return err
	}
	glog.Infoln("Exported blockchain to", fn)
	return nil
}
