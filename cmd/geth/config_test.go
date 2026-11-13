// Copyright 2026 The go-ethereum Authors
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
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfigRemovedFields checks that config files written by older
// releases still load when they contain fields that have since been removed.
func TestLoadConfigRemovedFields(t *testing.T) {
	for _, line := range []string{
		"EnableStateSizeTracking = true",
		"BinTrieGroupDepth = 5",
	} {
		file := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(file, []byte("[Eth]\n"+line+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		var cfg gethConfig
		if err := loadConfig(file, &cfg); err != nil {
			t.Errorf("%s: %v", line, err)
		}
	}
}
