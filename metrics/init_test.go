package metrics

import "time"

func init() {
	metricsEnabled = true
	MeterTickerInterval = 1 * time.Second
}
