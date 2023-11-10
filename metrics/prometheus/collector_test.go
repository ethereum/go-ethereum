package prometheus

import (
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
)

func TestMain(m *testing.M) {
	metrics.Enabled = true
	os.Exit(m.Run())
}

func TestCollector(t *testing.T) {
	c := newCollector()

	counter := metrics.NewCounter()
	counter.Inc(12345)
	c.addCounter("test/counter", counter)

	counterfloat64 := metrics.NewCounterFloat64()
	counterfloat64.Inc(54321.98)
	c.addCounterFloat64("test/counter_float64", counterfloat64)

	gauge := metrics.NewGauge()
	gauge.Update(23456)
	c.addGauge("test/gauge", gauge)

	gaugeFloat64 := metrics.NewGaugeFloat64()
	gaugeFloat64.Update(34567.89)
	c.addGaugeFloat64("test/gauge_float64", gaugeFloat64)

	histogram := metrics.NewHistogram(&metrics.NilSample{})
	c.addHistogram("test/histogram", histogram)

	meter := metrics.NewMeter()
	defer meter.Stop()
	meter.Mark(9999999)
	c.addMeter("test/meter", meter)

	timer := metrics.NewTimer()
	defer timer.Stop()
	timer.Update(20 * time.Millisecond)
	timer.Update(21 * time.Millisecond)
	timer.Update(22 * time.Millisecond)
	timer.Update(120 * time.Millisecond)
	timer.Update(23 * time.Millisecond)
	timer.Update(24 * time.Millisecond)
	c.addTimer("test/timer", timer)

	resettingTimer := metrics.NewResettingTimer()
	resettingTimer.Update(10 * time.Millisecond)
	resettingTimer.Update(11 * time.Millisecond)
	resettingTimer.Update(12 * time.Millisecond)
	resettingTimer.Update(120 * time.Millisecond)
	resettingTimer.Update(13 * time.Millisecond)
	resettingTimer.Update(14 * time.Millisecond)
	c.addResettingTimer("test/resetting_timer", resettingTimer.Snapshot())

	emptyResettingTimer := metrics.NewResettingTimer().Snapshot()
	c.addResettingTimer("test/empty_resetting_timer", emptyResettingTimer)

	const expectedOutput = `# TYPE test_counter gauge
test_counter 12345

# TYPE test_counter_float64 gauge
test_counter_float64 54321.98

# TYPE test_gauge gauge
test_gauge 23456

# TYPE test_gauge_float64 gauge
test_gauge_float64 34567.89

# TYPE test_histogram_count counter
test_histogram_count 0

# TYPE test_histogram summary
test_histogram {quantile="0.5"} 0
test_histogram {quantile="0.75"} 0
test_histogram {quantile="0.95"} 0
test_histogram {quantile="0.99"} 0
test_histogram {quantile="0.999"} 0
test_histogram {quantile="0.9999"} 0

# TYPE test_meter gauge
test_meter 9999999

# TYPE test_timer_count counter
test_timer_count 6

# TYPE test_timer summary
test_timer {quantile="0.5"} 2.25e+07
test_timer {quantile="0.75"} 4.8e+07
test_timer {quantile="0.95"} 1.2e+08
test_timer {quantile="0.99"} 1.2e+08
test_timer {quantile="0.999"} 1.2e+08
test_timer {quantile="0.9999"} 1.2e+08

# TYPE test_resetting_timer_count counter
test_resetting_timer_count 6

# TYPE test_resetting_timer summary
test_resetting_timer {quantile="0.50"} 12000000
test_resetting_timer {quantile="0.95"} 120000000
test_resetting_timer {quantile="0.99"} 120000000

`
	exp := c.buff.String()
	if exp != expectedOutput {
		t.Log("Expected Output:\n", expectedOutput)
		t.Log("Actual Output:\n", exp)
		t.Fatal("unexpected collector output")
	}
}
