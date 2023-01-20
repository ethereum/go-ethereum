package influxdb

import (
	"fmt"
	uurl "net/url"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/influxdata/influxdb/client"
)

type reporter struct {
	reg      metrics.Registry
	interval time.Duration

	url       uurl.URL
	database  string
	username  string
	password  string
	namespace string
	tags      map[string]string

	client *client.Client

	cache map[string]int64
}

// InfluxDB starts a InfluxDB reporter which will post the from the given metrics.Registry at each d interval.
func InfluxDB(r metrics.Registry, d time.Duration, url, database, username, password, namespace string) {
	InfluxDBWithTags(r, d, url, database, username, password, namespace, nil)
}

// InfluxDBWithTags starts a InfluxDB reporter which will post the from the given metrics.Registry at each d interval with the specified tags
func InfluxDBWithTags(r metrics.Registry, d time.Duration, url, database, username, password, namespace string, tags map[string]string) {
	u, err := uurl.Parse(url)
	if err != nil {
		log.Warn("Unable to parse InfluxDB", "url", url, "err", err)
		return
	}

	rep := &reporter{
		reg:       r,
		interval:  d,
		url:       *u,
		database:  database,
		username:  username,
		password:  password,
		namespace: namespace,
		tags:      tags,
		cache:     make(map[string]int64),
	}
	if err := rep.makeClient(); err != nil {
		log.Warn("Unable to make InfluxDB client", "err", err)
		return
	}

	rep.run()
}

// InfluxDBWithTagsOnce runs once an InfluxDB reporter and post the given metrics.Registry with the specified tags
func InfluxDBWithTagsOnce(r metrics.Registry, url, database, username, password, namespace string, tags map[string]string) error {
	u, err := uurl.Parse(url)
	if err != nil {
		return fmt.Errorf("unable to parse InfluxDB. url: %s, err: %v", url, err)
	}

	rep := &reporter{
		reg:       r,
		url:       *u,
		database:  database,
		username:  username,
		password:  password,
		namespace: namespace,
		tags:      tags,
		cache:     make(map[string]int64),
	}
	if err := rep.makeClient(); err != nil {
		return fmt.Errorf("unable to make InfluxDB client. err: %v", err)
	}

	if err := rep.send(); err != nil {
		return fmt.Errorf("unable to send to InfluxDB. err: %v", err)
	}

	return nil
}

func (r *reporter) makeClient() (err error) {
	r.client, err = client.NewClient(client.Config{
		URL:      r.url,
		Username: r.username,
		Password: r.password,
		Timeout:  10 * time.Second,
	})

	return
}

func (r *reporter) run() {
	intervalTicker := time.NewTicker(r.interval)
	pingTicker := time.NewTicker(time.Second * 5)

	defer intervalTicker.Stop()
	defer pingTicker.Stop()

	for {
		select {
		case <-intervalTicker.C:
			if err := r.send(); err != nil {
				log.Warn("Unable to send to InfluxDB", "err", err)
			}
		case <-pingTicker.C:
			_, _, err := r.client.Ping()
			if err != nil {
				log.Warn("Got error while sending a ping to InfluxDB, trying to recreate client", "err", err)

				if err = r.makeClient(); err != nil {
					log.Warn("Unable to make InfluxDB client", "err", err)
				}
			}
		}
	}
}

func (r *reporter) send() error {
	var pts []client.Point

	r.reg.Each(func(name string, i interface{}) {
		now := time.Now()
		namespace := r.namespace

		switch metric := i.(type) {
		case metrics.Counter:
			count := metric.Count()
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s%s.count", namespace, name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"value": count,
				},
				Time: now,
			})
		case metrics.Gauge:
			ms := metric.Snapshot()
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s%s.gauge", namespace, name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"value": ms.Value(),
				},
				Time: now,
			})
		case metrics.GaugeFloat64:
			ms := metric.Snapshot()
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s%s.gauge", namespace, name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"value": ms.Value(),
				},
				Time: now,
			})
		case metrics.Histogram:
			ms := metric.Snapshot()
			if ms.Count() > 0 {
				ps := ms.Percentiles([]float64{0.25, 0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
				fields := map[string]interface{}{
					"count":    ms.Count(),
					"max":      ms.Max(),
					"mean":     ms.Mean(),
					"min":      ms.Min(),
					"stddev":   ms.StdDev(),
					"variance": ms.Variance(),
					"p25":      ps[0],
					"p50":      ps[1],
					"p75":      ps[2],
					"p95":      ps[3],
					"p99":      ps[4],
					"p999":     ps[5],
					"p9999":    ps[6],
				}
				pts = append(pts, client.Point{
					Measurement: fmt.Sprintf("%s%s.histogram", namespace, name),
					Tags:        r.tags,
					Fields:      fields,
					Time:        now,
				})
			}
		case metrics.Meter:
			ms := metric.Snapshot()
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s%s.meter", namespace, name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"count": ms.Count(),
					"m1":    ms.Rate1(),
					"m5":    ms.Rate5(),
					"m15":   ms.Rate15(),
					"mean":  ms.RateMean(),
				},
				Time: now,
			})
		case metrics.Timer:
			ms := metric.Snapshot()
			ps := ms.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s%s.timer", namespace, name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
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
				},
				Time: now,
			})
		case metrics.ResettingTimer:
			t := metric.Snapshot()

			if len(t.Values()) > 0 {
				ps := t.Percentiles([]float64{50, 95, 99})
				val := t.Values()
				pts = append(pts, client.Point{
					Measurement: fmt.Sprintf("%s%s.span", namespace, name),
					Tags:        r.tags,
					Fields: map[string]interface{}{
						"count": len(val),
						"max":   val[len(val)-1],
						"mean":  t.Mean(),
						"min":   val[0],
						"p50":   ps[0],
						"p95":   ps[1],
						"p99":   ps[2],
					},
					Time: now,
				})
			}
		}
	})

	bps := client.BatchPoints{
		Points:   pts,
		Database: r.database,
	}

	_, err := r.client.Write(bps)
	return err
}
