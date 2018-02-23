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
	"github.com/ethereum/go-ethereum/log"
	gethmetrics "github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	"gopkg.in/urfave/cli.v1"
)

var (
	metricsInfluxDBEndpointFlag = cli.StringFlag{
		Name:  "metrics.influxdb.endpoint",
		Usage: "Metrics InfluxDB endpoint",
		Value: "http://127.0.0.1:8086",
	}
	metricsInfluxDBDatabaseFlag = cli.StringFlag{
		Name:  "metrics.influxdb.database",
		Usage: "metrics InfluxDB database",
		Value: "metrics",
	}
	metricsInfluxDBUsernameFlag = cli.StringFlag{
		Name:  "metrics.influxdb.username",
		Usage: "metrics InfluxDB username",
		Value: "",
	}
	metricsInfluxDBPasswordFlag = cli.StringFlag{
		Name:  "metrics.influxdb.password",
		Usage: "metrics InfluxDB password",
		Value: "",
	}
	// The `host` tag is part of every measurement sent to InfluxDB. Queries on tags are faster in InfluxDB.
	// It is used so that we can group all nodes and average a measurement across all of them, but also so
	// that we can select a specific node and inspect its measurements.
	// https://docs.influxdata.com/influxdb/v1.4/concepts/key_concepts/#tag-key
	metricsInfluxDBHostTagFlag = cli.StringFlag{
		Name:  "metrics.influxdb.host.tag",
		Usage: "metrics InfluxDB `host` tag attached to all measurements",
		Value: "localhost",
	}
)

// Flags holds all command-line flags required for metrics collection.
var Flags = []cli.Flag{
	utils.MetricsEnabledFlag,
	metricsInfluxDBEndpointFlag, metricsInfluxDBDatabaseFlag, metricsInfluxDBUsernameFlag, metricsInfluxDBPasswordFlag, metricsInfluxDBHostTagFlag,
}

func Setup(ctx *cli.Context) {
	if gethmetrics.Enabled {
		var (
			endpoint = ctx.GlobalString(metricsInfluxDBEndpointFlag.Name)
			database = ctx.GlobalString(metricsInfluxDBDatabaseFlag.Name)
			username = ctx.GlobalString(metricsInfluxDBUsernameFlag.Name)
			password = ctx.GlobalString(metricsInfluxDBPasswordFlag.Name)
			hosttag  = ctx.GlobalString(metricsInfluxDBHostTagFlag.Name)
		)

		log.Info("Enabling swarm metrics collection and export")
		go influxdb.InfluxDBWithTags(gethmetrics.DefaultRegistry, 10*time.Second, endpoint, database, username, password, "swarm.", map[string]string{
			"host": hosttag,
		})
	}
}
