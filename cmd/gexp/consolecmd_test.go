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
	"crypto/rand"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/expanse-org/go-expanse/params"
	"github.com/expanse-org/go-expanse/rpc"
)

// Tests that a node embedded within a console can be started up properly and
// then terminated by closing the input stream.
func TestConsoleWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"

	// Start a gexp console, make sure it's cleaned up and terminate the console
	gexp := runGeth(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--etherbase", coinbase, "--shh",
		"console")

	// Gather all the infos the welcome message needs to contain
	gexp.setTemplateFunc("goos", func() string { return runtime.GOOS })
	gexp.setTemplateFunc("gover", runtime.Version)
	gexp.setTemplateFunc("gethver", func() string { return params.Version })
	gexp.setTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	gexp.setTemplateFunc("apis", func() []string {
		apis := append(strings.Split(rpc.DefaultIPCApis, ","), rpc.MetadataApi)
		sort.Strings(apis)
		return apis
	})

	// Verify the actual welcome message to the required template
	gexp.expect(`
Welcome to the Gexp JavaScript console!

instance: Gexp/v{{gethver}}/{{goos}}/{{gover}}
coinbase: {{.Etherbase}}
at block: 0 ({{niltime}})
 datadir: {{.Datadir}}
 modules:{{range apis}} {{.}}:1.0{{end}}

> {{.InputLine "exit"}}
`)
	gexp.expectExit()
}

// Tests that a console can be attached to a running node via various means.
func TestIPCAttachWelcome(t *testing.T) {
	// Configure the instance for IPC attachement
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	var ipc string
	if runtime.GOOS == "windows" {
		ipc = `\\.\pipe\gexp` + strconv.Itoa(trulyRandInt(100000, 999999))
	} else {
		ws := tmpdir(t)
		defer os.RemoveAll(ws)
		ipc = filepath.Join(ws, "gexp.ipc")
	}
	// Note: we need --shh because testAttachWelcome checks for default
	// list of ipc modules and shh is included there.
	gexp := runGeth(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--etherbase", coinbase, "--shh", "--ipcpath", ipc)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gexp, "ipc:"+ipc)

	gexp.interrupt()
	gexp.expectExit()
}

func TestHTTPAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P
	gexp := runGeth(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--etherbase", coinbase, "--rpc", "--rpcport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gexp, "http://localhost:"+port)

	gexp.interrupt()
	gexp.expectExit()
}

func TestWSAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P

	gexp := runGeth(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--etherbase", coinbase, "--ws", "--wsport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gexp, "ws://localhost:"+port)

	gexp.interrupt()
	gexp.expectExit()
}

func testAttachWelcome(t *testing.T, gexp *testgeth, endpoint string) {
	// Attach to a running gexp note and terminate immediately
	attach := runGeth(t, "attach", endpoint)
	defer attach.expectExit()
	attach.stdin.Close()

	// Gather all the infos the welcome message needs to contain
	attach.setTemplateFunc("goos", func() string { return runtime.GOOS })
	attach.setTemplateFunc("gover", runtime.Version)
	attach.setTemplateFunc("gethver", func() string { return params.Version })
	attach.setTemplateFunc("etherbase", func() string { return gexp.Etherbase })
	attach.setTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	attach.setTemplateFunc("ipc", func() bool { return strings.HasPrefix(endpoint, "ipc") })
	attach.setTemplateFunc("datadir", func() string { return gexp.Datadir })
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

	// Verify the actual welcome message to the required template
	attach.expect(`
Welcome to the Gexp JavaScript console!

instance: Gexp/v{{gethver}}/{{goos}}/{{gover}}
coinbase: {{etherbase}}
at block: 0 ({{niltime}}){{if ipc}}
 datadir: {{datadir}}{{end}}
 modules:{{range apis}} {{.}}:1.0{{end}}

> {{.InputLine "exit" }}
`)
	attach.expectExit()
}

// trulyRandInt generates a crypto random integer used by the console tests to
// not clash network ports with other tests running cocurrently.
func trulyRandInt(lo, hi int) int {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(hi-lo)))
	return int(num.Int64()) + lo
}
