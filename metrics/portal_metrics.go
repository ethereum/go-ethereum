package metrics

import (
	"os"
	"path"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
)

type networkFileMetric struct {
	filename string
	metric   Gauge
	file     *os.File
	network  string
}

// CollectPortalMetrics periodically collects various metrics about system entities.
func CollectPortalMetrics(refresh time.Duration, networks []string, dataDir string) {
	// Short circuit if the metrics system is disabled
	if !Enabled {
		return
	}

	// Define the various metrics to collect
	var (
		historyTotalStorage = GetOrRegisterGauge("portal/history/total_storage", nil)
		beaconTotalStorage  = GetOrRegisterGauge("portal/beacon/total_storage", nil)
		stateTotalStorage   = GetOrRegisterGauge("portal/state/total_storage", nil)
	)

	var metricsArr []*networkFileMetric
	if slices.Contains(networks, portalwire.History.Name()) {
		dbPath := path.Join(dataDir, portalwire.History.Name())
		metricsArr = append(metricsArr, &networkFileMetric{
			filename: path.Join(dbPath, portalwire.History.Name()+".sqlite"),
			metric:   historyTotalStorage,
			network:  portalwire.History.Name(),
		})
	}
	if slices.Contains(networks, portalwire.Beacon.Name()) {
		dbPath := path.Join(dataDir, portalwire.Beacon.Name())
		metricsArr = append(metricsArr, &networkFileMetric{
			filename: path.Join(dbPath, portalwire.Beacon.Name()+".sqlite"),
			metric:   beaconTotalStorage,
			network:  portalwire.Beacon.Name(),
		})
	}
	if slices.Contains(networks, portalwire.State.Name()) {
		dbPath := path.Join(dataDir, portalwire.State.Name())
		metricsArr = append(metricsArr, &networkFileMetric{
			filename: path.Join(dbPath, portalwire.State.Name()+".sqlite"),
			metric:   stateTotalStorage,
			network:  portalwire.State.Name(),
		})
	}

	for {
		for _, m := range metricsArr {
			var err error = nil
			if m.file == nil {
				m.file, err = os.OpenFile(m.filename, os.O_RDONLY, 0600)
				if err != nil {
					log.Debug("Could not open file", "network", m.network, "file", m.filename, "metric", "total_storage", "err", err)
				}
			}
			if m.file != nil && err == nil {
				stat, err := m.file.Stat()
				if err != nil {
					log.Warn("Could not get file stat", "network", m.network, "file", m.filename, "metric", "total_storage", "err", err)
				}
				if err == nil {
					m.metric.Update(stat.Size())
				}
			}
		}

		time.Sleep(refresh)
	}
}
