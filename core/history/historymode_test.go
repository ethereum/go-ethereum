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

package history

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func TestNewPolicy(t *testing.T) {
	// KeepAll: no target.
	p, err := NewPolicy(KeepAll, params.MainnetGenesisHash, nil)
	if err != nil {
		t.Fatalf("KeepAll: %v", err)
	}
	if p.Mode != KeepAll || p.Target != nil {
		t.Errorf("KeepAll: unexpected policy %+v", p)
	}

	// PostMerge: resolves known mainnet prune point.
	p, err = NewPolicy(KeepPostMerge, params.MainnetGenesisHash, nil)
	if err != nil {
		t.Fatalf("PostMerge: %v", err)
	}
	if p.Target == nil || p.Target.BlockNumber != 15537393 {
		t.Errorf("PostMerge: unexpected target %+v", p.Target)
	}

	// PostPrague: resolves known mainnet prune point.
	p, err = NewPolicy(KeepPostPrague, params.MainnetGenesisHash, nil)
	if err != nil {
		t.Fatalf("PostPrague: %v", err)
	}
	if p.Target == nil || p.Target.BlockNumber != 22431084 {
		t.Errorf("PostPrague: unexpected target %+v", p.Target)
	}

	// PostMerge on unknown network: error.
	if _, err = NewPolicy(KeepPostMerge, common.HexToHash("0xdeadbeef"), nil); err == nil {
		t.Fatal("PostMerge unknown network: expected error")
	}
}

// customPoint is a real mainnet block, used to exercise the custom mode with a
// realistic pair.
var customPoint = &PrunePoint{
	BlockNumber: 25182208,
	BlockHash:   common.HexToHash("0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad"),
}

func TestNewPolicyCustom(t *testing.T) {
	// KeepCustom resolves the operator's own point, untouched.
	p, err := NewPolicy(KeepCustom, params.MainnetGenesisHash, customPoint)
	if err != nil {
		t.Fatalf("KeepCustom: %v", err)
	}
	if p.Mode != KeepCustom || p.Target == nil || *p.Target != *customPoint {
		t.Errorf("KeepCustom: unexpected policy %+v", p)
	}
	// The custom point is not looked up in staticPrunePoints, so it works on a
	// network that has no built-in entry for it.
	if _, err := NewPolicy(KeepCustom, common.HexToHash("0xdeadbeef"), customPoint); err != nil {
		t.Errorf("KeepCustom on unknown network: %v", err)
	}
	// Missing point: error, rather than a policy that prunes nothing.
	if _, err := NewPolicy(KeepCustom, params.MainnetGenesisHash, nil); err == nil {
		t.Error("KeepCustom without point: expected error")
	}
	// Degenerate points: error.
	for name, point := range map[string]*PrunePoint{
		"genesis":   {BlockNumber: 0, BlockHash: customPoint.BlockHash},
		"zero hash": {BlockNumber: customPoint.BlockNumber},
	} {
		if _, err := NewPolicy(KeepCustom, params.MainnetGenesisHash, point); err == nil {
			t.Errorf("KeepCustom with %s: expected error", name)
		}
	}
	// A point supplied for a mode that ignores it is a mistake worth reporting.
	for _, mode := range []HistoryMode{KeepAll, KeepPostMerge, KeepPostPrague, KeepPostOsaka} {
		if _, err := NewPolicy(mode, params.MainnetGenesisHash, customPoint); err == nil {
			t.Errorf("%s with a custom point: expected error", mode)
		}
	}
}

func TestHistoryModeText(t *testing.T) {
	for _, tt := range []struct {
		mode    HistoryMode
		text    string
		aliases []string
	}{
		{KeepAll, "all", nil},
		{KeepPostMerge, "postmerge", nil},
		{KeepPostPrague, "postprague", nil},
		{KeepPostOsaka, "postosaka", nil},
		{KeepCustom, "custom", nil},
	} {
		if got := tt.mode.String(); got != tt.text {
			t.Errorf("String of %d: got %q want %q", tt.mode, got, tt.text)
		}
		text, err := tt.mode.MarshalText()
		if err != nil || string(text) != tt.text {
			t.Errorf("MarshalText of %d: got %q, %v", tt.mode, text, err)
		}
		for _, name := range append([]string{tt.text}, tt.aliases...) {
			var got HistoryMode
			if err := got.UnmarshalText([]byte(name)); err != nil {
				t.Errorf("UnmarshalText(%q): %v", name, err)
				continue
			}
			if got != tt.mode {
				t.Errorf("UnmarshalText(%q): got mode %d want %d", name, got, tt.mode)
			}
		}
	}
	for _, bad := range []string{"", " ", "ALL", "post", "custom:25182208", "invalid HistoryMode(7)"} {
		var got HistoryMode
		if err := got.UnmarshalText([]byte(bad)); err == nil {
			t.Errorf("UnmarshalText(%q): expected error, got mode %d", bad, got)
		}
	}
	// The name list used for flag help must cover every valid mode, and every name in
	// it must resolve back to a distinct mode.
	names := HistoryModeNames()
	if len(names) != int(KeepCustom)+1 {
		t.Errorf("HistoryModeNames lists %d names for %d modes", len(names), int(KeepCustom)+1)
	}
	seen := make(map[HistoryMode]bool)
	for _, name := range names {
		var got HistoryMode
		if err := got.UnmarshalText([]byte(name)); err != nil {
			t.Errorf("HistoryModeNames lists %q which does not parse: %v", name, err)
			continue
		}
		if seen[got] {
			t.Errorf("HistoryModeNames lists %q twice for mode %d", name, got)
		}
		seen[got] = true
	}
	for mode := KeepAll; mode <= KeepCustom; mode++ {
		if !seen[mode] {
			t.Errorf("HistoryModeNames is missing mode %d (%q)", mode, mode)
		}
	}
}

func TestParsePrunePoint(t *testing.T) {
	got, err := ParsePrunePoint("25182208:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad")
	if err != nil {
		t.Fatalf("valid point: %v", err)
	}
	if *got != *customPoint {
		t.Errorf("valid point: got %+v want %+v", got, customPoint)
	}
	// String formats the point back into the form this parses, which is what the
	// "run prune-history again" hint relies on.
	again, err := ParsePrunePoint(got.String())
	if err != nil || *again != *got {
		t.Errorf("round trip through String: %+v, %v", again, err)
	}

	for _, bad := range []string{
		"",
		"25182208", // no hash
		":0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad",                     // no number
		"0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad",                      // hash only
		"0x17f4d20:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad",            // number must be decimal
		"-1:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad",                   // negative
		"18446744073709551616:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad", // overflows uint64
		"0:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad",                    // genesis
		"25182208:0xdeadbeef", // short hash
		"25182208:deadbeef00000000000000000000000000000000000000000000000000000000",         // missing 0x prefix
		"25182208:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aa",        // too short
		"25182208:0xzz7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad",       // not hex
		"25182208:0x0000000000000000000000000000000000000000000000000000000000000000",       // zero hash
		"25182208:0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad:extra", // trailing junk
		"25182208 :0x6f7c16414e091d817bdbb0e1d0a17f74cd2b42d1a734d9864a7cd37a32514aad",      // space in the number
	} {
		if got, err := ParsePrunePoint(bad); err == nil {
			t.Errorf("ParsePrunePoint(%q): expected error, got %+v", bad, got)
		}
	}
}

// TestChainHistoryText checks the text form shared by --history.chain and the
// config file, so that a dumped config can be fed back in unchanged.
func TestChainHistoryText(t *testing.T) {
	for _, mode := range []HistoryMode{KeepAll, KeepPostMerge, KeepPostPrague, KeepPostOsaka} {
		var got ChainHistory
		if err := got.UnmarshalText([]byte(mode.String())); err != nil || got != (ChainHistory{Mode: mode}) {
			t.Errorf("%q: got %+v, %v", mode, got, err)
		}
		if text, err := got.MarshalText(); err != nil || string(text) != mode.String() {
			t.Errorf("%q: marshalled as %q, %v", mode, text, err)
		}
	}
	value := customPoint.String()
	var got ChainHistory
	if err := got.UnmarshalText([]byte(value)); err != nil || got.Mode != KeepCustom || got.Point == nil || *got.Point != *customPoint {
		t.Fatalf("%q: got %+v, %v", value, got, err)
	}
	if text, err := got.MarshalText(); err != nil || string(text) != value {
		t.Errorf("custom point marshalled as %q, %v, want %q", text, err, value)
	}
	for _, bad := range []string{"", "custom", "postfoo", "25182208", "25182208:0xdeadbeef"} {
		var got ChainHistory
		if err := got.UnmarshalText([]byte(bad)); err == nil {
			t.Errorf("%q: expected error, got %+v", bad, got)
		}
	}
	if _, err := (ChainHistory{Mode: KeepCustom}).MarshalText(); err == nil {
		t.Error("custom retention without a point: expected marshal error")
	}
}

// TestStaticPrunePoints checks the built-in points are usable as configured: every
// mode that needs a point has entries, they are non-degenerate, and they resolve
// through the same paths a node uses.
func TestStaticPrunePoints(t *testing.T) {
	for mode := KeepPostMerge; mode <= KeepPostOsaka; mode++ {
		networks := staticPrunePoints[mode]
		if len(networks) == 0 {
			t.Errorf("%s has no prune points for any network", mode)
		}
		var parsed HistoryMode
		if err := parsed.UnmarshalText([]byte(mode.String())); err != nil {
			t.Errorf("%s: name %q does not parse: %v", mode, mode.String(), err)
		} else if parsed != mode {
			t.Errorf("%s: name %q parses to %d", mode, mode.String(), parsed)
		}
		for genesis, point := range networks {
			switch {
			case point == nil:
				t.Errorf("%s on %s: nil prune point", mode, genesis)
				continue
			case point.BlockNumber == 0:
				t.Errorf("%s on %s: prune point at genesis", mode, genesis)
			case point.BlockHash == (common.Hash{}):
				t.Errorf("%s on %s: prune point has a zero hash", mode, genesis)
			}
			p, err := NewPolicy(mode, genesis, nil)
			if err != nil {
				t.Errorf("%s on %s: %v", mode, genesis, err)
				continue
			}
			if p.Target != point {
				t.Errorf("%s on %s: target %+v, want %+v", mode, genesis, p.Target, point)
			}
		}
	}
}
