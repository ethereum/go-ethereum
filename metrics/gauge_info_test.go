// Copyright 2023 The CortexTheseus Authors
// This file is part of the CortexTheseus library.
//
// The CortexTheseus library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The CortexTheseus library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the CortexTheseus library. If not, see <http://www.gnu.org/licenses/>.

package metrics

import (
	"strconv"
	"testing"
)

func TestGaugeInfoJsonString(t *testing.T) {
	g := NewGaugeInfo()
	g.Update(GaugeInfoValue{
		"chain_id":   "5",
		"anotherKey": "any_string_value",
		"third_key":  "anything",
	},
	)
	want := `{"anotherKey":"any_string_value","chain_id":"5","third_key":"anything"}`
	if have := g.Value().String(); have != want {
		t.Errorf("\nhave: %v\nwant: %v\n", have, want)
	}
}

func TestGaugeInfoSnapshot(t *testing.T) {
	g := NewGaugeInfo()
	g.Update(GaugeInfoValue{"value": "original"})
	snapshot := g.Snapshot() // Snapshot @chainid 5
	g.Update(GaugeInfoValue{"value": "updated"})
	// The 'g' should be updated
	if have, want := g.Value().String(), `{"value":"updated"}`; have != want {
		t.Errorf("\nhave: %v\nwant: %v\n", have, want)
	}
	// Snapshot should be unupdated
	if have, want := snapshot.Value().String(), `{"value":"original"}`; have != want {
		t.Errorf("\nhave: %v\nwant: %v\n", have, want)
	}
}

func TestGetOrRegisterGaugeInfo(t *testing.T) {
	r := NewRegistry()
	NewRegisteredGaugeInfo("foo", r).Update(
		GaugeInfoValue{"chain_id": "5"})
	g := GetOrRegisterGaugeInfo("foo", r)
	if have, want := g.Value().String(), `{"chain_id":"5"}`; have != want {
		t.Errorf("have\n%v\nwant\n%v\n", have, want)
	}
}

func TestFunctionalGaugeInfo(t *testing.T) {
	info := GaugeInfoValue{"chain_id": "0"}
	counter := 1
	// A "functional" gauge invokes the method to obtain the value
	fg := NewFunctionalGaugeInfo(func() GaugeInfoValue {
		info["chain_id"] = strconv.Itoa(counter)
		counter++
		return info
	})
	fg.Value()
	fg.Value()
	if have, want := info["chain_id"], "2"; have != want {
		t.Errorf("have %v want %v", have, want)
	}
}

func TestGetOrRegisterFunctionalGaugeInfo(t *testing.T) {
	r := NewRegistry()
	NewRegisteredFunctionalGaugeInfo("foo", r, func() GaugeInfoValue {
		return GaugeInfoValue{
			"chain_id": "5",
		}
	})
	want := `{"chain_id":"5"}`
	have := GetOrRegisterGaugeInfo("foo", r).Value().String()
	if have != want {
		t.Errorf("have\n%v\nwant\n%v\n", have, want)
	}
}
