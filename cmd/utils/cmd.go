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
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

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
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/peterh/liner"
)

const (
	importBatchSize = 2500
	legalese        = `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse a tincidunt magna. Phasellus a eros volutpat, sagittis ipsum sit amet, eleifend quam. Aenean venenatis ultricies feugiat. Nulla finibus arcu blandit tincidunt rutrum. Aliquam maximus convallis elementum. Etiam ornare molestie tortor, quis scelerisque est laoreet et. Sed lobortis pellentesque metus, et bibendum libero efficitur quis. Sed posuere sapien erat, vitae tempus neque maximus tincidunt. Nam fermentum lectus in scelerisque convallis. In laoreet volutpat enim, eget laoreet nulla vehicula iaculis. Pellentesque vel mattis lorem. Fusce consectetur orci at bibendum fermentum. Vestibulum venenatis vitae ipsum vel rhoncus. Nulla facilisi. Donec imperdiet, eros a eleifend dignissim, mauris lacus pharetra arcu, et aliquam lacus enim a magna. Phasellus congue consectetur tellus a vehicula.

Praesent laoreet quis leo et lacinia. Cras a laoreet orci. Quisque magna nisl, dignissim eget aliquet ut, bibendum mattis justo. Fusce at tortor ligula. Nulla sollicitudin mollis euismod. Nulla enim sem, interdum ac auctor non, faucibus id risus. Duis nisi mauris, maximus vel ex ut, ullamcorper vehicula arcu. Sed nec lobortis nibh. Sed malesuada semper nulla sit amet tristique. Fusce at leo orci. Quisque nec porttitor ante. Nunc scelerisque dolor lectus, iaculis auctor mi mattis id. Donec tempor non tellus id ultricies. Praesent at felis non augue auctor efficitur.

Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Quisque cursus ullamcorper dapibus. Suspendisse fringilla erat eget nunc dapibus pellentesque eget eget ante. Morbi sollicitudin nec ex eget finibus. Nam volutpat nunc at elit varius, id fringilla lectus sollicitudin. Curabitur ac varius ex. Nam commodo nibh a neque aliquam fringilla. Morbi suscipit magna sit amet enim tincidunt sollicitudin. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae;

Ut pretium iaculis pellentesque. Nam eros tortor, malesuada a varius nec, aliquet placerat magna. Integer rutrum porttitor cursus. Praesent in pharetra turpis, eget fringilla neque. Aliquam venenatis tellus lectus, nec imperdiet nibh accumsan vel. Maecenas semper dapibus velit, ac pretium tortor. Maecenas dapibus, nunc sit amet egestas porttitor, arcu ipsum maximus lorem, non varius lorem turpis eget tortor. Cras at purus aliquam, blandit nunc placerat, imperdiet tellus. Phasellus dignissim venenatis dictum. Aliquam eu nisi nibh. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Suspendisse sit amet ultrices metus, at pulvinar eros. Suspendisse sollicitudin posuere metus sed pulvinar. Cras et velit vel sem gravida faucibus quis quis mi. Vivamus eleifend ante sit amet ultricies tincidunt.

Mauris et elementum nulla. Fusce at scelerisque purus. Proin molestie sapien id velit viverra, a pharetra quam tempor. Fusce orci risus, semper et interdum at, imperdiet eget lectus. Praesent feugiat ante ut egestas tempor. Morbi convallis, quam sed mattis consequat, libero diam interdum sem, quis tempor enim nibh a ligula. Quisque est felis, pharetra nec pharetra vel, euismod et tellus. Nulla et dui nulla. Aliquam consectetur nunc ligula, sed molestie odio elementum vitae. Mauris neque nisi, venenatis et est ut, vehicula accumsan lectus.

Do you accept this agreement?`
)

var interruptCallbacks = []func(os.Signal){}

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

	if liner.TerminalSupported() {
		lr := liner.NewLiner()
		defer lr.Close()
		input, err = lr.Prompt(prompt)
	} else {
		fmt.Print(prompt)
		input, err = bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Println()
	}

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

func CheckLegalese(datadir string) {
	// check "first run"
	if !common.FileExist(datadir) {
		r, _ := PromptConfirm(legalese)
		if !r {
			Fatalf("Must accept to continue. Shutting down...\n")
		}
	}
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

func StartEthereum(ethereum *eth.Ethereum) {
	glog.V(logger.Info).Infoln("Starting", ethereum.Name())
	if err := ethereum.Start(); err != nil {
		Fatalf("Error starting Ethereum: %v", err)
	}
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)
		defer signal.Stop(sigc)
		<-sigc
		glog.V(logger.Info).Infoln("Got interrupt, shutting down...")
		go ethereum.Stop()
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

func ImportChain(chain *core.ChainManager, fn string) error {
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

func hasAllBlocks(chain *core.ChainManager, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash()) {
			return false
		}
	}
	return true
}

func ExportChain(chainmgr *core.ChainManager, fn string) error {
	glog.Infoln("Exporting blockchain to", fn)
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := chainmgr.Export(fh); err != nil {
		return err
	}
	glog.Infoln("Exported blockchain to", fn)
	return nil
}

func ExportAppendChain(chainmgr *core.ChainManager, fn string, first uint64, last uint64) error {
	glog.Infoln("Exporting blockchain to", fn)
	// TODO verify mode perms
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := chainmgr.ExportN(fh, first, last); err != nil {
		return err
	}
	glog.Infoln("Exported blockchain to", fn)
	return nil
}
