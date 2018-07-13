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

// stateth is a command to orchestrate influxdb/grafana setup using
// docker containers in order to visualize stats from Geth for
// development purposes.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/log"
	gapi "github.com/teemupo/go-grafana-api"
	"gopkg.in/urfave/cli.v1"
)

var (
	dockerPrefix     string // unique prefix used for the created docker resources
	dockerNetwork    string // name of the docker network to start containers in
	dashboardsFolder string // folder containing all dashboards to be imported in Grafana
	grafanaPort      int    // expose port for the Grafana HTTP interface
	influxdbPort     int    // expose port for the InfluxDB HTTP interface
	influxdbDatabase string // name of database to provision on InfluxDB
	rm               bool   // remove stateth containers and stateth network upon startup
)

const (
	influxdbAdminUser = "test"  // admin username for InfluxDB
	influxdbAdminPass = "test"  // admin password for InfluxDB
	grafanaUser       = "admin" // default Grafana username - should not be changed here without first updating the docker image
	grafanaPass       = "admin" // default Grafana password - should not be changed here without first udpating the docker image
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
			Name:  "influxdb-database",
			Value: "geth",
			Usage: "default influxdb database to provision",
		},
		cli.StringFlag{
			Name:  "grafana-dashboards-folder",
			Value: os.Getenv("GOPATH") + "/src/github.com/ethereum/go-ethereum/cmd/stateth/grafana_dashboards",
			Usage: "default grafana dashboards folder",
		},
		cli.StringFlag{
			Name:  "docker-prefix",
			Value: "stateth",
			Usage: "prefix to be used for docker network (if not specified explicitly) and containers. must be unique.",
		},
		cli.StringFlag{
			Name:  "docker-network",
			Value: "",
			Usage: "docker network to start containers in.",
		},
		cli.BoolFlag{
			Name:  "rm",
			Usage: "remove existing stateth network and stateth containers upon startup.",
		},
	}
	app.Action = func(c *cli.Context) error {
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(c.Int("loglevel")), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

		dockerPrefix = c.String("docker-prefix")
		dockerNetwork = c.String("docker-network")
		if dockerNetwork == "" {
			dockerNetwork = dockerPrefix
		}
		grafanaPort = c.Int("grafana-http-port")
		influxdbPort = c.Int("influxdb-http-port")
		dashboardsFolder = c.String("grafana-dashboards-folder")
		influxdbDatabase = c.String("influxdb-database")
		rm = c.Bool("rm")

		if rm {
			cleanupContainers(c)
		}

		runNetwork(c)

		if err := runInfluxDB(c); err != nil {
			return err
		}
		if err := runGrafana(c); err != nil {
			return err
		}
		log.Info("waiting for grafana to boot up...")
		time.Sleep(7 * time.Second) // give time to Grafana to boot up
		if err := importGrafanaDatasource(c); err != nil {
			return err
		}
		if err := importGrafanaDashboards(c); err != nil {
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

		fmt.Println(fmt.Sprintf("grafana listening on http://localhost:%d", grafanaPort))
		fmt.Println(fmt.Sprintf("username: %s", grafanaUser))
		fmt.Println(fmt.Sprintf("password: %s", grafanaPass))
		fmt.Println()
		fmt.Println("waiting for SIGINT or SIGTERM (CTRL^C) to stop service and remove containers...")
		<-done

		cleanupContainers(c)

		return nil
	}

	app.Run(os.Args)
}

func runNetwork(c *cli.Context) {
	log.Info("creating docker network", "network", dockerNetwork)
	command := strings.Split(fmt.Sprintf("docker network create %s", dockerNetwork), " ")
	r, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Warn(string(r))
	}
}

func runInfluxDB(c *cli.Context) error {
	log.Info("pulling influxdb:1.5.2 docker image")
	command := strings.Split("docker pull influxdb:1.5.2", " ")
	r, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("running influxdb docker container", "container", fmt.Sprintf("%s_influxdb", dockerPrefix))
	command = strings.Split(fmt.Sprintf("docker run --network %s --name %s_influxdb -e INFLUXDB_DB=%s -e INFLUXDB_ADMIN_USER=%s -e INFLUXDB_ADMIN_PASSWORD=%s -p %d:8086 -d influxdb:1.5.2", dockerNetwork, dockerPrefix, influxdbDatabase, influxdbAdminUser, influxdbAdminPass, influxdbPort), " ")
	r, err = exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Error(string(r))
		return err
	}
	return nil
}

func runGrafana(c *cli.Context) error {
	log.Info("pulling grafana/grafana:5.1.3 docker image")
	command := strings.Split("docker pull grafana/grafana:5.1.3", " ")
	r, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Error(string(r))
		return err
	}

	log.Info("running grafana docker container", "container", fmt.Sprintf("%s_grafana", dockerPrefix))
	command = strings.Split(fmt.Sprintf("docker run --network %s --name=%s_grafana -p %d:3000 -e GF_AUTH_ANONYMOUS_ENABLED=true -d grafana/grafana:5.1.3", dockerNetwork, dockerPrefix, grafanaPort), " ")
	r, err = exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Error(string(r))
		return err
	}
	return nil
}

func cleanupContainers(c *cli.Context) {
	log.Info("removing influxdb container")
	command := strings.Split(fmt.Sprintf("docker rm -f %s_influxdb", dockerPrefix), " ")
	r, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Warn(string(r))
	}

	log.Info("removing grafana container")
	command = strings.Split(fmt.Sprintf("docker rm -f %s_grafana", dockerPrefix), " ")
	r, err = exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Warn(string(r))
	}

	log.Info("removing network")
	command = strings.Split(fmt.Sprintf("docker network rm %s", dockerNetwork), " ")
	r, err = exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Warn(string(r))
	}
}

func importGrafanaDatasource(c *cli.Context) error {
	log.Info("importing grafana datasource")
	gclient, err := gapi.New(fmt.Sprintf("%s:%s", grafanaUser, grafanaPass), fmt.Sprintf("http://localhost:%d", grafanaPort))
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	dataSource := &gapi.DataSource{
		Name:      influxdbDatabase,
		Type:      "influxdb",
		URL:       fmt.Sprintf("http://%s_influxdb:%d", dockerPrefix, influxdbPort),
		Access:    "proxy",
		Database:  influxdbDatabase,
		User:      influxdbAdminUser,
		Password:  influxdbAdminPass,
		IsDefault: true,
		BasicAuth: false,
	}

	_, err = gclient.NewDataSource(dataSource)
	if err != nil {
		log.Warn(err.Error())
		return err
	}

	return nil
}

func importGrafanaDashboards(c *cli.Context) error {
	log.Info("importing grafana dashboards")
	gclient, err := gapi.New(fmt.Sprintf("%s:%s", grafanaUser, grafanaPass), fmt.Sprintf("http://localhost:%d", grafanaPort))
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	files, err := ioutil.ReadDir(dashboardsFolder)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	for _, f := range files {
		name := f.Name()
		if strings.Contains(name, "json") {
			log.Info("importing dashboard", "dashboard", name)

			blob, err := ioutil.ReadFile(filepath.Join(dashboardsFolder, name))
			if err != nil {
				log.Warn(err.Error())
				return nil
			}

			model := prepareDashboardModel(string(blob))

			_, err = gclient.SaveDashboard(model, false)
			if err != nil {
				log.Warn(err.Error())
				return nil
			}

		}
	}

	return nil
}

func prepareDashboardModel(configJSON string) map[string]interface{} {
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		panic("invalid JSON got into prepare func")
	}

	delete(configMap, "id")
	// Only exists in 5.0+
	delete(configMap, "uid")
	configMap["version"] = 0

	return configMap
}
