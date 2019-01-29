// Copyright 2018 The go-ethereum Authors
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

package metrics

import (
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	gethmetrics "github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	"github.com/ethereum/go-ethereum/swarm/log"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	MetricsEnableInfluxDBExportFlag = cli.BoolFlag{
		Name:  "metrics.influxdb.export",
		Usage: "Enable metrics export/push to an external InfluxDB database",
	}
	MetricsEnableInfluxDBAccountingExportFlag = cli.BoolFlag{
		Name:  "metrics.influxdb.accounting",
		Usage: "Enable accounting metrics export/push to an external InfluxDB database",
	}
	MetricsInfluxDBEndpointFlag = cli.StringFlag{
		Name:  "metrics.influxdb.endpoint",
		Usage: "Metrics InfluxDB endpoint",
		Value: "http://127.0.0.1:8086",
	}
	MetricsInfluxDBDatabaseFlag = cli.StringFlag{
		Name:  "metrics.influxdb.database",
		Usage: "Metrics InfluxDB database",
		Value: "metrics",
	}
	MetricsInfluxDBUsernameFlag = cli.StringFlag{
		Name:  "metrics.influxdb.username",
		Usage: "Metrics InfluxDB username",
		Value: "",
	}
	MetricsInfluxDBPasswordFlag = cli.StringFlag{
		Name:  "metrics.influxdb.password",
		Usage: "Metrics InfluxDB password",
		Value: "",
	}
	// Tags are part of every measurement sent to InfluxDB. Queries on tags are faster in InfluxDB.
	// For example `host` tag could be used so that we can group all nodes and average a measurement
	// across all of them, but also so that we can select a specific node and inspect its measurements.
	// https://docs.influxdata.com/influxdb/v1.4/concepts/key_concepts/#tag-key
	MetricsInfluxDBTagsFlag = cli.StringFlag{
		Name:  "metrics.influxdb.tags",
		Usage: "Comma-separated InfluxDB tags (key/values) attached to all measurements",
		Value: "host=localhost",
	}
)

// Flags holds all command-line flags required for metrics collection.
var Flags = []cli.Flag{
	utils.MetricsEnabledFlag,
	MetricsEnableInfluxDBExportFlag,
	MetricsEnableInfluxDBAccountingExportFlag,
	MetricsInfluxDBEndpointFlag,
	MetricsInfluxDBDatabaseFlag,
	MetricsInfluxDBUsernameFlag,
	MetricsInfluxDBPasswordFlag,
	MetricsInfluxDBTagsFlag,
}

func Setup(ctx *cli.Context) {
	if gethmetrics.Enabled {
		log.Info("Enabling swarm metrics collection")
		var (
			endpoint               = ctx.GlobalString(MetricsInfluxDBEndpointFlag.Name)
			database               = ctx.GlobalString(MetricsInfluxDBDatabaseFlag.Name)
			username               = ctx.GlobalString(MetricsInfluxDBUsernameFlag.Name)
			password               = ctx.GlobalString(MetricsInfluxDBPasswordFlag.Name)
			enableExport           = ctx.GlobalBool(MetricsEnableInfluxDBExportFlag.Name)
			enableAccountingExport = ctx.GlobalBool(MetricsEnableInfluxDBAccountingExportFlag.Name)
		)

		// Start system runtime metrics collection
		go gethmetrics.CollectProcessMetrics(2 * time.Second)

		tagsMap := utils.SplitTagsFlag(ctx.GlobalString(MetricsInfluxDBTagsFlag.Name))

		if enableExport {
			log.Info("Enabling swarm metrics export to InfluxDB")
			go influxdb.InfluxDBWithTags(gethmetrics.DefaultRegistry, 10*time.Second, endpoint, database, username, password, "swarm.", tagsMap)
		}

		if enableAccountingExport {
			log.Info("Exporting swarm accounting metrics to InfluxDB")
			go influxdb.InfluxDBWithTags(gethmetrics.AccountingRegistry, 10*time.Second, endpoint, database, username, password, "accounting.", tagsMap)
		}
	}
}
