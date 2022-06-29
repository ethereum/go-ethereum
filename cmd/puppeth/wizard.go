// Copyright 2017 The go-ethereum Authors
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
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/peterh/liner"
	"golang.org/x/term"
)

// config contains all the configurations needed by puppeth that should be saved
// between sessions.
type config struct {
	path      string   // File containing the configuration values
	bootnodes []string // Bootnodes to always connect to by all nodes
	ethstats  string   // Ethstats settings to cache for node deploys

	Genesis *core.Genesis     `json:"genesis,omitempty"` // Genesis block to cache for node deploys
	Servers map[string][]byte `json:"servers,omitempty"`
}

// servers retrieves an alphabetically sorted list of servers.
func (c config) servers() []string {
	servers := make([]string, 0, len(c.Servers))
	for server := range c.Servers {
		servers = append(servers, server)
	}
	sort.Strings(servers)

	return servers
}

// flush dumps the contents of config to disk.
func (c config) flush() {
	os.MkdirAll(filepath.Dir(c.path), 0755)

	out, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(c.path, out, 0644); err != nil {
		log.Warn("Failed to save puppeth configs", "file", c.path, "err", err)
	}
}

type wizard struct {
	network string // Network name to manage
	conf    config // Configurations from previous runs

	servers  map[string]*sshClient // SSH connections to servers to administer
	services map[string][]string   // Ethereum services known to be running on servers

	lock sync.Mutex // Lock to protect configs during concurrent service discovery
}

// prompts the user for input with the given prompt string.  Returns when a value is entered.
// Causes the wizard to exit if ctrl-d is pressed
func promptInput(p string) string {
	for {
		text, err := prompt.Stdin.PromptInput(p)
		if err != nil {
			if err != liner.ErrPromptAborted {
				log.Crit("Failed to read user input", "err", err)
			}
		} else {
			return text
		}
	}
}

// read reads a single line from stdin, trimming if from spaces.
func (w *wizard) read() string {
	text := promptInput("> ")
	return strings.TrimSpace(text)
}

// readString reads a single line from stdin, trimming if from spaces, enforcing
// non-emptyness.
func (w *wizard) readString() string {
	for {
		text := promptInput("> ")
		if text = strings.TrimSpace(text); text != "" {
			return text
		}
	}
}

// readDefaultString reads a single line from stdin, trimming if from spaces. If
// an empty line is entered, the default value is returned.
func (w *wizard) readDefaultString(def string) string {
	text := promptInput("> ")
	if text = strings.TrimSpace(text); text != "" {
		return text
	}
	return def
}

// readDefaultYesNo reads a single line from stdin, trimming if from spaces and
// interpreting it as a 'yes' or a 'no'. If an empty line is entered, the default
// value is returned.
func (w *wizard) readDefaultYesNo(def bool) bool {
	for {
		text := promptInput("> ")
		if text = strings.ToLower(strings.TrimSpace(text)); text == "" {
			return def
		}
		if text == "y" || text == "yes" {
			return true
		}
		if text == "n" || text == "no" {
			return false
		}
		log.Error("Invalid input, expected 'y', 'yes', 'n', 'no' or empty")
	}
}

// readURL reads a single line from stdin, trimming if from spaces and trying to
// interpret it as a URL (http, https or file).
func (w *wizard) readURL() *url.URL {
	for {
		text := promptInput("> ")
		uri, err := url.Parse(strings.TrimSpace(text))
		if err != nil {
			log.Error("Invalid input, expected URL", "err", err)
			continue
		}
		return uri
	}
}

// readInt reads a single line from stdin, trimming if from spaces, enforcing it
// to parse into an integer.
func (w *wizard) readInt() int {
	for {
		text := promptInput("> ")
		if text = strings.TrimSpace(text); text == "" {
			continue
		}
		val, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil {
			log.Error("Invalid input, expected integer", "err", err)
			continue
		}
		return val
	}
}

// readDefaultInt reads a single line from stdin, trimming if from spaces, enforcing
// it to parse into an integer. If an empty line is entered, the default value is
// returned.
func (w *wizard) readDefaultInt(def int) int {
	for {
		text := promptInput("> ")
		if text = strings.TrimSpace(text); text == "" {
			return def
		}
		val, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil {
			log.Error("Invalid input, expected integer", "err", err)
			continue
		}
		return val
	}
}

// readDefaultBigInt reads a single line from stdin, trimming if from spaces,
// enforcing it to parse into a big integer. If an empty line is entered, the
// default value is returned.
func (w *wizard) readDefaultBigInt(def *big.Int) *big.Int {
	for {
		text := promptInput("> ")
		if text = strings.TrimSpace(text); text == "" {
			return def
		}
		val, ok := new(big.Int).SetString(text, 0)
		if !ok {
			log.Error("Invalid input, expected big integer")
			continue
		}
		return val
	}
}

// readDefaultFloat reads a single line from stdin, trimming if from spaces, enforcing
// it to parse into a float. If an empty line is entered, the default value is returned.
func (w *wizard) readDefaultFloat(def float64) float64 {
	for {
		text := promptInput("> ")
		if text = strings.TrimSpace(text); text == "" {
			return def
		}
		val, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			log.Error("Invalid input, expected float", "err", err)
			continue
		}
		return val
	}
}

// readPassword reads a single line from stdin, trimming it from the trailing new
// line and returns it. The input will not be echoed.
func (w *wizard) readPassword() string {
	fmt.Printf("> ")
	text, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Crit("Failed to read password", "err", err)
	}
	fmt.Println()
	return string(text)
}

// readAddress reads a single line from stdin, trimming if from spaces and converts
// it to an Ethereum address.
func (w *wizard) readAddress() *common.Address {
	for {
		text := promptInput("> 0x")
		if text = strings.TrimSpace(text); text == "" {
			return nil
		}
		// Make sure it looks ok and return it if so
		if len(text) != 40 {
			log.Error("Invalid address length, please retry")
			continue
		}
		bigaddr, _ := new(big.Int).SetString(text, 16)
		address := common.BigToAddress(bigaddr)
		return &address
	}
}

// readDefaultAddress reads a single line from stdin, trimming if from spaces and
// converts it to an Ethereum address. If an empty line is entered, the default
// value is returned.
func (w *wizard) readDefaultAddress(def common.Address) common.Address {
	for {
		// Read the address from the user
		text := promptInput("> 0x")
		if text = strings.TrimSpace(text); text == "" {
			return def
		}
		// Make sure it looks ok and return it if so
		if len(text) != 40 {
			log.Error("Invalid address length, please retry")
			continue
		}
		bigaddr, _ := new(big.Int).SetString(text, 16)
		return common.BigToAddress(bigaddr)
	}
}

// readJSON reads a raw JSON message and returns it.
func (w *wizard) readJSON() string {
	var blob json.RawMessage

	for {
		text := promptInput("> ")
		reader := strings.NewReader(text)
		if err := json.NewDecoder(reader).Decode(&blob); err != nil {
			log.Error("Invalid JSON, please try again", "err", err)
			continue
		}
		return string(blob)
	}
}

// readIPAddress reads a single line from stdin, trimming if from spaces and
// returning it if it's convertible to an IP address. The reason for keeping
// the user input format instead of returning a Go net.IP is to match with
// weird formats used by ethstats, which compares IPs textually, not by value.
func (w *wizard) readIPAddress() string {
	for {
		// Read the IP address from the user
		fmt.Printf("> ")
		text := promptInput("> ")
		if text = strings.TrimSpace(text); text == "" {
			return ""
		}
		// Make sure it looks ok and return it if so
		if ip := net.ParseIP(text); ip == nil {
			log.Error("Invalid IP address, please retry")
			continue
		}
		return text
	}
}
