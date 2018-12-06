// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"net/http/httptrace"
	"os"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	gethmetrics "github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	swarmmetrics "github.com/ethereum/go-ethereum/swarm/metrics"
	"github.com/ethereum/go-ethereum/swarm/tracing"

	"github.com/ethereum/go-ethereum/log"

	cli "gopkg.in/urfave/cli.v1"
)

var (
	gitCommit string // Git SHA1 commit hash of the release (set via linker flags)
)

var (
	endpoints        []string
	includeLocalhost bool
	cluster          string
	appName          string
	scheme           string
	filesize         int
	from             int
	to               int
	verbosity        int
	timeout          int
)

func main() {

	app := cli.NewApp()
	app.Name = "smoke-test"
	app.Usage = ""

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "cluster-endpoint",
			Value:       "prod",
			Usage:       "cluster to point to (prod or a given namespace)",
			Destination: &cluster,
		},
		cli.StringFlag{
			Name:        "app",
			Value:       "swarm",
			Usage:       "application to point to (swarm or swarm-private)",
			Destination: &appName,
		},
		cli.IntFlag{
			Name:        "cluster-from",
			Value:       8501,
			Usage:       "swarm node (from)",
			Destination: &from,
		},
		cli.IntFlag{
			Name:        "cluster-to",
			Value:       8512,
			Usage:       "swarm node (to)",
			Destination: &to,
		},
		cli.StringFlag{
			Name:        "cluster-scheme",
			Value:       "http",
			Usage:       "http or https",
			Destination: &scheme,
		},
		cli.BoolFlag{
			Name:        "include-localhost",
			Usage:       "whether to include localhost:8500 as an endpoint",
			Destination: &includeLocalhost,
		},
		cli.IntFlag{
			Name:        "filesize",
			Value:       1024,
			Usage:       "file size for generated random file in KB",
			Destination: &filesize,
		},
		cli.IntFlag{
			Name:        "verbosity",
			Value:       1,
			Usage:       "verbosity",
			Destination: &verbosity,
		},
		cli.IntFlag{
			Name:        "timeout",
			Value:       120,
			Usage:       "timeout in seconds after which kill the process",
			Destination: &timeout,
		},
	}

	app.Flags = append(app.Flags, []cli.Flag{
		utils.MetricsEnabledFlag,
		swarmmetrics.MetricsInfluxDBEndpointFlag,
		swarmmetrics.MetricsInfluxDBDatabaseFlag,
		swarmmetrics.MetricsInfluxDBUsernameFlag,
		swarmmetrics.MetricsInfluxDBPasswordFlag,
		swarmmetrics.MetricsInfluxDBHostTagFlag,
	}...)

	app.Flags = append(app.Flags, tracing.Flags...)

	app.Commands = []cli.Command{
		{
			Name:    "upload_and_sync",
			Aliases: []string{"c"},
			Usage:   "upload and sync",
			Action:  cliUploadAndSync,
		},
		{
			Name:    "feed_sync",
			Aliases: []string{"f"},
			Usage:   "feed update generate, upload and sync",
			Action:  cliFeedUploadAndSync,
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Before = func(ctx *cli.Context) error {
		tracing.Setup(ctx)
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		return emitMetrics(ctx)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error(err.Error())

		os.Exit(1)
	}
}

func emitMetrics(ctx *cli.Context) error {
	if gethmetrics.Enabled {
		var (
			endpoint = ctx.GlobalString(swarmmetrics.MetricsInfluxDBEndpointFlag.Name)
			database = ctx.GlobalString(swarmmetrics.MetricsInfluxDBDatabaseFlag.Name)
			username = ctx.GlobalString(swarmmetrics.MetricsInfluxDBUsernameFlag.Name)
			password = ctx.GlobalString(swarmmetrics.MetricsInfluxDBPasswordFlag.Name)
			hosttag  = ctx.GlobalString(swarmmetrics.MetricsInfluxDBHostTagFlag.Name)
		)
		return influxdb.InfluxDBWithTagsOnce(gethmetrics.DefaultRegistry, endpoint, database, username, password, "swarm-smoke.", map[string]string{
			"host":     hosttag,
			"version":  gitCommit,
			"filesize": fmt.Sprintf("%v", filesize),
		})
	}

	return nil
}

func getClientTrace(testName, ruid string, tn *time.Time) *httptrace.ClientTrace {
	trace := &httptrace.ClientTrace{
		GetConn: func(_ string) {
			log.Trace(testName+" - http get", "event", "GetConn", "ruid", ruid)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.getconn", nil).Update(time.Since(*tn))
		},
		GotConn: func(_ httptrace.GotConnInfo) {
			log.Trace(testName+" - http get", "event", "GotConn", "ruid", ruid)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.gotconn", nil).Update(time.Since(*tn))
		},
		PutIdleConn: func(err error) {
			log.Trace(testName+" - http get", "event", "PutIdleConn", "ruid", ruid, "err", err)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.putidle", nil).Update(time.Since(*tn))
		},
		GotFirstResponseByte: func() {
			log.Trace(testName+" - http get", "event", "GotFirstResponseByte", "ruid", ruid)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.firstbyte", nil).Update(time.Since(*tn))
		},
		Got100Continue: func() {
			log.Trace(testName+" - http get", "event", "Got100Continue", "ruid", ruid)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.got100continue", nil).Update(time.Since(*tn))
		},
		DNSStart: func(_ httptrace.DNSStartInfo) {
			log.Trace(testName+" - http get", "event", "DNSStart", "ruid", ruid)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.dnsstart", nil).Update(time.Since(*tn))
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			log.Trace(testName+" - http get", "event", "DNSDone", "ruid", ruid)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.dnsdone", nil).Update(time.Since(*tn))
		},
		ConnectStart: func(network, addr string) {
			log.Trace(testName+" - http get", "event", "ConnectStart", "ruid", ruid, "network", network, "addr", addr)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.connectstart", nil).Update(time.Since(*tn))
		},
		ConnectDone: func(network, addr string, err error) {
			log.Trace(testName+" - http get", "event", "ConnectDone", "ruid", ruid, "network", network, "addr", addr, "err", err)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.connectdone", nil).Update(time.Since(*tn))
		},
		WroteHeaders: func() {
			log.Trace(testName+" - http get", "event", "WroteHeaders(request)", "ruid", ruid)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.wroteheaders", nil).Update(time.Since(*tn))
		},
		Wait100Continue: func() {
			log.Trace(testName+" - http get", "event", "Wait100Continue", "ruid", ruid)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.wait100continue", nil).Update(time.Since(*tn))
		},
		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			log.Trace(testName+" - http get", "event", "WroteRequest", "ruid", ruid)
			gethmetrics.GetOrRegisterResettingTimer(testName+".fetch.clienttrace.wroterequest", nil).Update(time.Since(*tn))
		},
	}
	return trace
}
