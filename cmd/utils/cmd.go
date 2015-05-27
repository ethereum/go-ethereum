/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 * 	Viktor Tron <viktor@ethdev.com>
 */
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

var interruptCallbacks = []func(os.Signal){}

// Register interrupt handlers callbacks
func RegisterInterrupt(cb func(os.Signal)) {
	interruptCallbacks = append(interruptCallbacks, cb)
}

// go routine that call interrupt handlers in order of registering
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	go func() {
		signal.Notify(c, os.Interrupt)
		for sig := range c {
			glog.V(logger.Error).Infof("Shutting down (%v) ... \n", sig)
			RunInterruptCallbacks(sig)
		}
	}()
}

func RunInterruptCallbacks(sig os.Signal) {
	for _, cb := range interruptCallbacks {
		cb(sig)
	}
}

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

func initDataDir(Datadir string) {
	_, err := os.Stat(Datadir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Data directory '%s' doesn't exist, creating it\n", Datadir)
			os.Mkdir(Datadir, 0777)
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
	RegisterInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		logger.Flush()
	})
}

func StartEthereumForTest(ethereum *eth.Ethereum) {
	glog.V(logger.Info).Infoln("Starting ", ethereum.Name())
	ethereum.StartForTest()
	RegisterInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		logger.Flush()
	})
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

	// Remove all existing blocks and start the import.
	chain.Reset()
	batchSize := 2500
	blocks := make(types.Blocks, batchSize)
	n := 0
	for {
		// Load a batch of RLP blocks.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		i := 0
		for ; i < batchSize; i++ {
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
		if _, err := chain.InsertChain(blocks[:i]); err != nil {
			return fmt.Errorf("invalid block %d: %v", n, err)
		}
	}
	return nil
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
