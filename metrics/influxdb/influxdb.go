package influxdb

import (
	"fmt"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/influxdata/influxdb-client-go/v2"
)

type reporter struct {
	reg      metrics.Registry
	interval time.Duration

	endpoint     string
	token        string
	bucket       string
	organization string
	namespace    string
	tags         map[string]string

	client influxdb2.Client
	write  api.WriteAPI

	cache map[string]int64
}

// InfluxDBWithTags starts a InfluxDB reporter which will post the from the given metrics.Registry at each d interval with the specified tags
func InfluxDBWithTags(r metrics.Registry, d time.Duration, endpoint string, token string, bucket string, organization string, namespace string, tags map[string]string) {
	rep := &reporter{
		reg:          r,
		interval:     d,
		endpoint:     endpoint,
		token:        token,
		bucket:       bucket,
		organization: organization,
		namespace:    namespace,
		tags:         tags,
		cache:        make(map[string]int64),
	}

	rep.client = influxdb2.NewClient(rep.endpoint, rep.token)
	// non-blocking write client
	rep.write = rep.client.WriteAPI(rep.organization, rep.bucket)

	// Listening for errors
	errorsCh := rep.write.Errors()
	// Create go proc for reading and logging errors
	go func() {
		for err := range errorsCh {
			fmt.Printf("write error: %s\n", err.Error())
		}
	}()
}

func (r *reporter) send() {
	r.reg.Each(func(name string, i interface{}) {
		now := time.Now()
		namespace := r.namespace

		switch metric := i.(type) {

		case metrics.Counter:
			v := metric.Count()
			l := r.cache[name]

			measurement := fmt.Sprintf("%s%s.count", namespace, name)
			fields := map[string]interface{}{
				"value": v - l,
			}

			pt := influxdb2.NewPoint(measurement, r.tags, fields, now)
			r.write.WritePoint(pt)

			r.cache[name] = v

		case metrics.Gauge:
			ms := metric.Snapshot()

			measurement := fmt.Sprintf("%s%s.gauge", namespace, name)
			fields := map[string]interface{}{
				"value": ms.Value(),
			}

			pt := influxdb2.NewPoint(measurement, r.tags, fields, now)
			r.write.WritePoint(pt)

		case metrics.GaugeFloat64:
			ms := metric.Snapshot()

			measurement := fmt.Sprintf("%s%s.gauge", namespace, name)
			fields := map[string]interface{}{
				"value": ms.Value(),
			}

			pt := influxdb2.NewPoint(measurement, r.tags, fields, now)
			r.write.WritePoint(pt)

		case metrics.Histogram:
			ms := metric.Snapshot()

			if ms.Count() > 0 {
				ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
				measurement := fmt.Sprintf("%s%s.histogram", namespace, name)
				fields := map[string]interface{}{
					"count":    ms.Count(),
					"max":      ms.Max(),
					"mean":     ms.Mean(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
				}

				pt := influxdb2.NewPoint(measurement, r.tags, fields, now)
				r.write.WritePoint(pt)
			}

		case metrics.Meter:
			ms := metric.Snapshot()

			measurement := fmt.Sprintf("%s%s.meter", namespace, name)
			fields := map[string]interface{}{
				"count": ms.Count(),
				"m1":    ms.Rate1(),
				"m5":    ms.Rate5(),
				"m15":   ms.Rate15(),
				"mean":  ms.RateMean(),
			}

			pt := influxdb2.NewPoint(measurement, r.tags, fields, now)
			r.write.WritePoint(pt)

		case metrics.Timer:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})

			measurement := fmt.Sprintf("%s%s.timer", namespace, name)
			fields := map[string]interface{}{
				"count":    ms.Count(),
				"max":      ms.Max(),
				"mean":     ms.Mean(),
				"min":      ms.Min(),
				"stddev":   ms.StdDev(),
				"variance": ms.Variance(),
				"p50":      ps[0],
				"p75":      ps[1],
				"p95":      ps[2],
				"p99":      ps[3],
				"p999":     ps[4],
				"p9999":    ps[5],
				"m1":       ms.Rate1(),
				"m5":       ms.Rate5(),
				"m15":      ms.Rate15(),
				"meanrate": ms.RateMean(),
			}

			pt := influxdb2.NewPoint(measurement, r.tags, fields, now)
			r.write.WritePoint(pt)

		case metrics.ResettingTimer:
			t := metric.Snapshot()

			if len(t.Values()) > 0 {
				ps := t.Percentiles([]float64{50, 95, 99})
				val := t.Values()

				measurement := fmt.Sprintf("%s%s.span", namespace, name)
				fields := map[string]interface{}{
					"count": len(val),
					"max":   val[len(val)-1],
					"mean":  t.Mean(),
					"min":   val[0],
					"p50":   ps[0],
					"p95":   ps[1],
					"p99":   ps[2],
				}

				pt := influxdb2.NewPoint(measurement, r.tags, fields, now)
				r.write.WritePoint(pt)
			}
		}
	})

	// Force all unwritten data to be sent
	r.write.Flush()
}
