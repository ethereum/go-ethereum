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

// TestHistoryTailTomL checks both directions for the custom pruning point, which is
// configured from a config file as well as from --history.tail.
//
// Marshalling is worth a test because the generated MarshalTOML copies fields one by
// one: declaring a field in its intermediate struct without also assigning it drops
// the value silently. The point also has to serialise as a string rather than a
// table, since a TOML table cannot be followed by plain keys.
func TestHistoryTailTomL(t *testing.T) {
	want := &history.PrunePoint{
		BlockNumber: 25182208,
		BlockHash:   common.HexToHash("0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad"),
	}
	// Marshalling: the generated MarshalTOML copies fields one by one, so a field
	// that is only declared in the intermediate struct is dropped without error.
	cfg := Defaults
	cfg.HistoryMode = history.KeepCustom
	cfg.HistoryTail = want
	// Wrapped by value, as the command's config struct embeds it; marshalling a
	// *Config directly would not consult MarshalTOML at all.
	out, err := toml.Marshal(struct{ Eth Config }{cfg})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if want := `history_tail = "25182208:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad"`; !strings.Contains(string(out), want) {
		t.Errorf("marshalled config is missing %q:\n%s", want, out)
	}
	// A config without a custom point must not grow the key.
	if out, err = toml.Marshal(struct{ Eth Config }{Defaults}); err != nil {
		t.Fatalf("marshal defaults: %v", err)
	}
	if strings.Contains(string(out), "history_tail") {
		t.Error("defaults gained a history_tail entry")
	}

	// Unmarshalling: a dumped config can be read back, keys as the marshaller writes
	// them, with the point in the same text form --history.tail takes.
	doc := "history_mode = \"custom\"\n" +
		"history_tail = \"25182208:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad\"\n"
	var got Config
	if err := toml.Unmarshal([]byte(doc), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.HistoryMode != history.KeepCustom {
		t.Errorf("history mode: got %q, want %q", got.HistoryMode, history.KeepCustom)
	}
	if got.HistoryTail == nil || *got.HistoryTail != *want {
		t.Errorf("history tail: got %+v, want %+v", got.HistoryTail, want)
	}
	// A malformed point is reported rather than ignored.
	var broken Config
	if err := toml.Unmarshal([]byte("history_tail = \"latest\"\n"), &broken); err == nil {
		t.Error("malformed HistoryTail: expected error")
	}
}
