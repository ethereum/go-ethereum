package metrics

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func ExampleOpenTSDB() {
	addr, _ := net.ResolveTCPAddr("net", ":2003")
	go OpenTSDB(DefaultRegistry, 1*time.Second, "some.prefix", addr)
}

func ExampleOpenTSDBWithConfig() {
	addr, _ := net.ResolveTCPAddr("net", ":2003")
	go OpenTSDBWithConfig(OpenTSDBConfig{
		Addr:          addr,
		Registry:      DefaultRegistry,
		FlushInterval: 1 * time.Second,
		DurationUnit:  time.Millisecond,
	})
}

func TestExampleOpenTSB(t *testing.T) {
	r := NewOrderedRegistry()
	NewRegisteredGaugeInfo("foo", r).Update(GaugeInfoValue{"chain_id": "5"})
	NewRegisteredGaugeFloat64("pi", r).Update(3.14)
	NewRegisteredCounter("months", r).Inc(12)
	NewRegisteredCounterFloat64("tau", r).Inc(1.57)
	NewRegisteredMeter("elite", r).Mark(1337)
	NewRegisteredTimer("second", r).Update(time.Second)
	NewRegisteredCounterFloat64("tau", r).Inc(1.57)
	NewRegisteredCounterFloat64("tau", r).Inc(1.57)

	w := new(strings.Builder)
	(&OpenTSDBConfig{
		Registry:     r,
		DurationUnit: time.Millisecond,
		Prefix:       "pre",
	}).writeRegistry(w, 978307200, "hal9000")

	wantB, err := os.ReadFile("./testdata/opentsb.want")
	if err != nil {
		t.Fatal(err)
	}
	if have, want := w.String(), string(wantB); have != want {
		t.Errorf("\nhave:\n%v\nwant:\n%v\n", have, want)
	}
}
