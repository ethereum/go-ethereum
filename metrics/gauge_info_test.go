package metrics

import (
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkGuageInfo(b *testing.B) {
	g := NewGaugeInfo()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Update(GaugeInfoValue{
			{"chain_id", string(rune(i))},
		})
	}
}

func TestGaugeInfo(t *testing.T) {
	g := NewGaugeInfo()
	g.Update(GaugeInfoValue{
		{"chain_id", "5"},
	},
	)
	expected := GaugeInfoValue{
		{"chain_id", "5"},
	}
	for idx, v := range g.Value() {
		if v.Key != expected[idx].Key || v.Val != expected[idx].Val {
			t.Errorf("g.Value()[%v]: %v != %v\n", idx, v, expected[idx])
		}
	}
}

func TestGaugeInfoSnapshot(t *testing.T) {
	g := NewGaugeInfo()
	g.Update(GaugeInfoValue{
		{"chain_id", "5"},
	})
	snapshot := g.Snapshot()
	g.Update(GaugeInfoValue{
		{"chain_id", "1"},
	})
	expected := GaugeInfoValue{
		{"chain_id", "5"},
	}
	for idx, v := range snapshot.Value() {
		if v.Key != expected[idx].Key || v.Val != expected[idx].Val {
			t.Errorf("g.Value()[%v]: %v != %v\n", idx, v, expected[idx])
		}
	}
}

func TestGetOrRegisterGaugeInfo(t *testing.T) {
	r := NewRegistry()
	NewRegisteredGaugeInfo("foo", r).Update(GaugeInfoValue{
		{"chain_id", "5"},
	})
	expected := GaugeInfoValue{
		{"chain_id", "5"},
	}
	g := GetOrRegisterGaugeInfo("foo", r)
	for idx, v := range g.Value() {
		if v.Key != expected[idx].Key || v.Val != expected[idx].Val {
			t.Fatal(g)
		}
	}
}

func TestFunctionalGaugeInfo(t *testing.T) {
	info := GaugeInfoValue{
		{"chain_id", "0"},
	}
	counter := 1
	fg := NewFunctionalGaugeInfo(func() GaugeInfoValue {
		info[0].Val = strconv.Itoa(counter)
		counter++
		return info
	})
	fg.Value()
	fg.Value()
	if info[0].Val != "2" {
		t.Error("info[0].Val != \"2\" -> ", info[0].Val)
	}
}

func TestGetOrRegisterFunctionalGaugeInfo(t *testing.T) {
	r := NewRegistry()
	NewRegisteredFunctionalGaugeInfo("foo", r, func() GaugeInfoValue {
		return GaugeInfoValue{
			{"chain_id", "5"},
		}
	})
	expected := GaugeInfoValue{
		{"chain_id", "5"},
	}
	g := GetOrRegisterGaugeInfo("foo", r)
	for idx, v := range g.Value() {
		if v.Key != expected[idx].Key || v.Val != expected[idx].Val {
			t.Fatal(g)
		}
	}
}

func TestGaugeInfoValueJsonString(t *testing.T) {
	g := NewGaugeInfo()
	g.Update(GaugeInfoValue{
		{"chain_id", "5"},
		{"anotherKey", "any_string_value"},
		{"third_key", "anything"},
	},
	)
	expected := `{"chain_id":"5","anotherKey":"any_string_value","third_key":"anything"}`
	got := g.ValueJsonString()
	if got != expected {
		t.Errorf("g.ValueToJsonString(): %s != %s\n", got, expected)
	}
}

func ExampleGetOrRegisterGaugeInfo() {
	m := "chain/info"
	g := GetOrRegisterGaugeInfo(m, nil)
	g.Update(GaugeInfoValue{
		{"chain_id", "5"},
		{"random_value", "10"},
		{"chain_data", "356"},
	})
	fmt.Println(g.Value()) // Output: [{chain_id 5} {random_value 10} {chain_data 356}]
}
