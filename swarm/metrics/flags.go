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
	metrics "github.com/ethersphere/go-metrics"
	influxdb "github.com/ethersphere/go-metrics-influxdb"
	"gopkg.in/urfave/cli.v1"
)

var (
	metricsEndpointFlag = cli.StringFlag{
		Name:  "metricsendpoint",
		Usage: "metrics backend endpoint",
		Value: "http://127.0.0.1:8086",
	}
	metricsDatabaseFlag = cli.StringFlag{
		Name:  "metricsdatabase",
		Usage: "metrics backend database",
		Value: "metrics",
	}
	metricsUsernameFlag = cli.StringFlag{
		Name:  "metricsusername",
		Usage: "metrics backend username",
		Value: "admin",
	}
	metricsPasswordFlag = cli.StringFlag{
		Name:  "metricspassword",
		Usage: "metrics backend password",
		Value: "admin",
	}
	metricsHostTagFlag = cli.StringFlag{
		Name:  "metricshosttag",
		Usage: "metrics host tag",
		Value: "localhost",
	}
)

// Flags holds all command-line flags required for metrics collection.
var Flags = []cli.Flag{
	utils.MetricsEnabledFlag,
	metricsEndpointFlag, metricsDatabaseFlag, metricsUsernameFlag, metricsPasswordFlag, metricsHostTagFlag,
}

func Setup(ctx *cli.Context) {
	if gethmetrics.Enabled {
		var (
			endpoint = ctx.GlobalString(metricsEndpointFlag.Name)
			database = ctx.GlobalString(metricsDatabaseFlag.Name)
			username = ctx.GlobalString(metricsUsernameFlag.Name)
			password = ctx.GlobalString(metricsPasswordFlag.Name)
			hosttag  = ctx.GlobalString(metricsHostTagFlag.Name)
		)

		log.Info("Enabling swarm metrics collection and export")
		go influxdb.InfluxDBWithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, database, username, password, "swarm.", map[string]string{
			"host": hosttag,
		})
	}
}
