// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethconfig

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/history"
	"github.com/naoina/toml"
)

// TestHistoryModeTOML checks that the chain history retention round trips through
// the config file in the same text form --history.chain takes, including a custom
// prune point, and that configs written before custom points existed still load.
func TestHistoryModeTOML(t *testing.T) {
	const value = "25182208:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad"
	want := history.ChainHistory{
		Mode: history.KeepCustom,
		Point: &history.PrunePoint{
			BlockNumber: 25182208,
			BlockHash:   common.HexToHash("0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad"),
		},
	}
	cfg := Defaults
	cfg.HistoryMode = want

	// Wrapped by value, as the command's config struct embeds it; marshalling a
	// *Config directly would not consult MarshalTOML at all.
	out, err := toml.Marshal(struct{ Eth Config }{cfg})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if line := `history_mode = "` + value + `"`; !strings.Contains(string(out), line) {
		t.Errorf("marshalled config is missing %q:\n%s", line, out)
	}
	var got Config
	if err := toml.Unmarshal([]byte(`history_mode = "`+value+`"`), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.HistoryMode.Mode != want.Mode || got.HistoryMode.Point == nil || *got.HistoryMode.Point != *want.Point {
		t.Errorf("got %+v, want %+v", got.HistoryMode, want)
	}
	var old Config
	if err := toml.Unmarshal([]byte(`history_mode = "postmerge"`), &old); err != nil || old.HistoryMode != (history.ChainHistory{Mode: history.KeepPostMerge}) {
		t.Errorf("named mode: got %+v, %v", old.HistoryMode, err)
	}
	var broken Config
	if err := toml.Unmarshal([]byte(`history_mode = "custom"`), &broken); err == nil {
		t.Error("custom without a point: expected error")
	}
}
