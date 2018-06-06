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

// puppeth is a command to assemble and maintain private networks.
package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/stateth"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	app := cli.NewApp()
	app.Name = "stateth"
	app.Usage = "run a local grafana/influxdb setup for local Geth node stats visualization"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "loglevel",
			Value: 3,
			Usage: "log level to emit to the screen",
		},
		cli.IntFlag{
			Name:  "influxdb-http-port",
			Value: 8086,
			Usage: "default influxdb http port",
		},
		cli.IntFlag{
			Name:  "grafana-http-port",
			Value: 3000,
			Usage: "default grafana http port",
		},
		cli.StringFlag{
			Name:  "grafana-dashboards-folder",
			Value: os.Getenv("GOPATH") + "/src/github.com/ethereum/go-ethereum/cmd/stateth/grafana_dashboards",
			Usage: "default grafana dashboards folder",
		},
		cli.StringFlag{
			Name:  "docker-prefix",
			Value: "stateth",
			Usage: "prefix to be used for docker network and containers. must be unique.",
		},
		cli.BoolFlag{
			Name:  "rm",
			Usage: "Remove existing stateth network and stateth containers upon startup. make sure that the start up works every time, even if the service wasn't shut down gracefully",
		},
	}
	app.Action = func(ctx *cli.Context) error {
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(ctx.Int("loglevel")), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

		se, err := stateth.New(ctx, &stateth.Config{
			DockerPrefix:     ctx.String("docker-prefix"),
			GrafanaPort:      ctx.Int("grafana-http-port"),
			InfluxDBPort:     ctx.Int("influxdb-http-port"),
			DashboardsFolder: ctx.String("grafana-dashboards-folder"),
			Rm:               ctx.Bool("rm"),
		})
		if err != nil {
			return err
		}
		if err = se.StartExternal(); err != nil {
			return err
		}

		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			sig := <-sigs
			fmt.Println()
			fmt.Println(sig)
			done <- true
		}()

		fmt.Println("waiting for SIGINT or SIGTERM (CTRL^C) to stop service and remove containers...")
		<-done

		return se.StopExternal()
	}

	app.Run(os.Args)
}
