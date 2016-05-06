// Copyright 2016 The go-ethereum Authors
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

package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/rpc"
)

// Tests that a node embedded within a console can be started up properly and
// then terminated by closing the input stream.
func TestConsoleWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"

	// Start a geth console, make sure it's cleaned up and terminate the console
	geth := runGeth(t, "--nat", "none", "--nodiscover", "--etherbase", coinbase, "-shh", "console")
	defer geth.expectExit()
	geth.stdin.Close()

	// Gather all the infos the welcome message needs to contain
	geth.setTemplateFunc("goos", func() string { return runtime.GOOS })
	geth.setTemplateFunc("gover", runtime.Version)
	geth.setTemplateFunc("gethver", func() string { return verString })
	geth.setTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	geth.setTemplateFunc("apis", func() []string {
		apis := append(strings.Split(rpc.DefaultIPCApis, ","), rpc.MetadataApi)
		sort.Strings(apis)
		return apis
	})
	geth.setTemplateFunc("prompt", func() string { return console.DefaultPrompt })

	// Verify the actual welcome message to the required template
	geth.expect(`
Welcome to the Geth JavaScript console!

instance: Geth/v{{gethver}}/{{goos}}/{{gover}}
coinbase: {{.Etherbase}}
at block: 0 ({{niltime}})
 datadir: {{.Datadir}}
 modules:{{range apis}} {{.}}:1.0{{end}}

{{prompt}}
`)
}

// Tests that a console can be attached to a running node via various means.
func TestIPCAttachWelcome(t *testing.T) {
	// Configure the instance for IPC attachement
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"

	var ipc string
	if runtime.GOOS == "windows" {
		ipc = `\\.\pipe\geth` + strconv.Itoa(rand.Int())
	} else {
		ws := tmpdir(t)
		defer os.RemoveAll(ws)

		ipc = filepath.Join(ws, "geth.ipc")
	}
	// Run the parent geth and attach with a child console
	geth := runGeth(t, "--nat", "none", "--nodiscover", "--etherbase", coinbase, "-shh", "--ipcpath", ipc)
	defer geth.interrupt()

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, geth, "ipc:"+ipc)
}

func TestHTTPAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(rand.Intn(65535-1024) + 1024) // Yeah, sometimes this will fail, sorry :P

	geth := runGeth(t, "--nat", "none", "--nodiscover", "--etherbase", coinbase, "--rpc", "--rpcport", port)
	defer geth.interrupt()

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, geth, "http://localhost:"+port)
}

func TestWSAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(rand.Intn(65535-1024) + 1024) // Yeah, sometimes this will fail, sorry :P

	geth := runGeth(t, "--nat", "none", "--nodiscover", "--etherbase", coinbase, "--ws", "--wsport", port)
	defer geth.interrupt()

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, geth, "ws://localhost:"+port)
}

func testAttachWelcome(t *testing.T, geth *testgeth, endpoint string) {
	// Attach to a running geth note and terminate immediately
	attach := runGeth(t, "attach", endpoint)
	defer attach.expectExit()
	attach.stdin.Close()

	// Gather all the infos the welcome message needs to contain
	attach.setTemplateFunc("goos", func() string { return runtime.GOOS })
	attach.setTemplateFunc("gover", runtime.Version)
	attach.setTemplateFunc("gethver", func() string { return verString })
	attach.setTemplateFunc("etherbase", func() string { return geth.Etherbase })
	attach.setTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	attach.setTemplateFunc("ipc", func() bool { return strings.HasPrefix(endpoint, "ipc") })
	attach.setTemplateFunc("datadir", func() string { return geth.Datadir })
	attach.setTemplateFunc("apis", func() []string {
		var apis []string
		if strings.HasPrefix(endpoint, "ipc") {
			apis = append(strings.Split(rpc.DefaultIPCApis, ","), rpc.MetadataApi)
		} else {
			apis = append(strings.Split(rpc.DefaultHTTPApis, ","), rpc.MetadataApi)
		}
		sort.Strings(apis)
		return apis
	})
	attach.setTemplateFunc("prompt", func() string { return console.DefaultPrompt })

	// Verify the actual welcome message to the required template
	attach.expect(`
Welcome to the Geth JavaScript console!

instance: Geth/v{{gethver}}/{{goos}}/{{gover}}
coinbase: {{etherbase}}
at block: 0 ({{niltime}}){{if ipc}}
 datadir: {{datadir}}{{end}}
 modules:{{range apis}} {{.}}:1.0{{end}}

{{prompt}}
`)
}
