package metrics

import (
	"database/sql"
	"errors"
	"os"
	"path"
	"slices"
	"strings"
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

type PortalStorageMetrics struct {
	RadiusRatio         GaugeFloat64
	EntriesCount        Gauge
	ContentStorageUsage Gauge
}

const (
	countEntrySql          = "SELECT COUNT(1) FROM kvstore;"
	contentStorageUsageSql = "SELECT SUM( length(value) ) FROM kvstore;"
)

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

func NewPortalStorageMetrics(network string, db *sql.DB) (*PortalStorageMetrics, error) {
	if !Enabled {
		return nil, nil
	}

	if network != portalwire.History.Name() && network != portalwire.Beacon.Name() && network != portalwire.State.Name() {
		log.Debug("Unknow network for metrics", "network", network)
		return nil, errors.New("unknow network for metrics")
	}

	var countSql string
	var contentSql string
	if network == portalwire.Beacon.Name() {
		countSql = strings.Replace(countEntrySql, "kvstore", "beacon", 1)
		contentSql = strings.Replace(contentStorageUsageSql, "kvstore", "beacon", 1)
		contentSql = strings.Replace(contentSql, "value", "content_value", 1)
	} else {
		countSql = countEntrySql
		contentSql = contentStorageUsageSql
	}

	storageMetrics := &PortalStorageMetrics{}

	storageMetrics.RadiusRatio = NewRegisteredGaugeFloat64("portal/"+network+"/radius_ratio", nil)
	storageMetrics.RadiusRatio.Update(1)

	storageMetrics.EntriesCount = NewRegisteredGauge("portal/"+network+"/entry_count", nil)
	log.Debug("Counting entities in " + network + " storage for metrics")
	var res *int64 = new(int64)
	q := db.QueryRow(countSql)
	if q.Err() == sql.ErrNoRows {
		storageMetrics.EntriesCount.Update(0)
	} else if q.Err() != nil {
		log.Error("Querry execution error", "network", network, "metric", "entry_count", "err", q.Err())
		return nil, q.Err()
	} else {
		q.Scan(res)
		storageMetrics.EntriesCount.Update(*res)
	}

	storageMetrics.ContentStorageUsage = NewRegisteredGauge("portal/"+network+"/content_storage", nil)
	log.Debug("Counting storage usage (bytes) in " + network + " for metrics")
	var res2 *int64 = new(int64)
	q2 := db.QueryRow(contentSql)
	if q2.Err() == sql.ErrNoRows {
		storageMetrics.ContentStorageUsage.Update(0)
	} else if q2.Err() != nil {
		log.Error("Querry execution error", "network", network, "metric", "entry_count", "err", q2.Err())
		return nil, q2.Err()
	} else {
		q2.Scan(res2)
		storageMetrics.ContentStorageUsage.Update(*res2)
	}

	return storageMetrics, nil
}
