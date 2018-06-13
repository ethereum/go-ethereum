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

package stateth

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	gapi "github.com/teemupo/go-grafana-api"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	influxDBAdminUser = "test"  // admin username for InfluxDB.
	influxDBAdminPass = "test"  // admin password for InfluxDB.
	grafanaUser       = "admin" // default Grafana username - should not be changed here without first updating the docker image.
	grafanaPass       = "admin" // default Grafana password - should not be changed here without first udpating the docker image.
)

// Stateth contains the stateth internals.
type Stateth struct {
	ctx     *cli.Context
	config  *Config
	gclient *gapi.Client // Grafana client.
}

// New creates a new stateth instance with the given configuration.
func New(ctx *cli.Context, config *Config) *Stateth {
	return &Stateth{
		ctx:    ctx,
		config: config,
	}
}

// Protocols implements the node.Service interface.
func (se *Stateth) Protocols() []p2p.Protocol { return nil }

// APIs implements the node.Service interface.
func (se *Stateth) APIs() []rpc.API { return nil }

// Start starts InfluxDB and Grafana.
// Implements the node.Service interface.
func (se *Stateth) Start(server *p2p.Server) error {
	return se.StartExternal(true)
}

// Stop cleans up the containers.
// Implements the node.Service interface.
func (se *Stateth) Stop() error {
	return se.StopExternal()
}

func (se *Stateth) StartExternal(rm bool) error {
	var err error
	if rm {
		se.cleanupContainers()
	}
	if err = se.runNetwork(); err != nil {
		return err
	}
	if err = se.runInfluxDB(); err != nil {
		return err
	}
	if err = se.runGrafana(); err != nil {
		return err
	}
	log.Info("Waiting for Grafana to boot up...")
	time.Sleep(7 * time.Second) // give time to Grafana to boot up

	se.gclient, err = gapi.New(fmt.Sprintf("%s:%s", grafanaUser, grafanaPass), fmt.Sprintf("http://localhost:%d", se.config.GrafanaPort))
	if err != nil {
		log.Warn(err.Error())
		return nil
	}
	if err = se.importGrafanaDatasource(); err != nil {
		return err
	}
	if err = se.importGrafanaDashboards(); err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("Grafana listening on http://localhost:%d", se.config.GrafanaPort))
	fmt.Println(fmt.Sprintf("Username: %s", grafanaUser))
	fmt.Println(fmt.Sprintf("Password: %s", grafanaPass))
	fmt.Println()

	return nil
}

func (se *Stateth) StopExternal() error {
	se.cleanupContainers()
	return nil
}

func (se *Stateth) runNetwork() error {
	log.Info("Creating docker network", "network", se.config.DockerPrefix)
	command := strings.Split(fmt.Sprintf("docker network create %s", se.config.DockerPrefix), " ")
	r, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Error(string(r))
		return err
	}
	return nil
}

func (se *Stateth) runInfluxDB() error {
	log.Info("Pulling influxdb:1.5.2 docker image")
	command := strings.Split("docker pull influxdb:1.5.2", " ")
	r, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("Running InfluxDB docker container", "container", fmt.Sprintf("%s_influxdb", se.config.DockerPrefix))
	command = strings.Split(fmt.Sprintf("docker run --network %s --name %s_influxdb -e INFLUXDB_DB=metrics -e INFLUXDB_ADMIN_USER=%s -e INFLUXDB_ADMIN_PASSWORD=%s -p %d:8086 -d influxdb:1.5.2", se.config.DockerPrefix, se.config.DockerPrefix, influxDBAdminUser, influxDBAdminPass, se.config.InfluxDBPort), " ")
	r, err = exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Error(string(r))
		return err
	}
	return nil
}

func (se *Stateth) runGrafana() error {
	log.Info("Pulling grafana/grafana:5.1.3 docker image")
	command := strings.Split("docker pull grafana/grafana:5.1.3", " ")
	r, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Error(string(r))
		return err
	}

	log.Info("Running Grafana docker container", "container", fmt.Sprintf("%s_grafana", se.config.DockerPrefix))
	command = strings.Split(fmt.Sprintf("docker run --network %s --name=%s_grafana -e GF_AUTH_ANONYMOUS_ENABLED=true -p %d:3000 -d grafana/grafana:5.1.3", se.config.DockerPrefix, se.config.DockerPrefix, se.config.GrafanaPort), " ")
	r, err = exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Error(string(r))
		return err
	}
	return nil
}

func (se *Stateth) cleanupContainers() {
	log.Info("Removing InfluxDB container")
	command := strings.Split(fmt.Sprintf("docker rm -f %s_influxdb", se.config.DockerPrefix), " ")
	r, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Warn(string(r))
	}

	log.Info("Removing Grafana container")
	command = strings.Split(fmt.Sprintf("docker rm -f %s_grafana", se.config.DockerPrefix), " ")
	r, err = exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Warn(string(r))
	}

	log.Info("Removing network")
	command = strings.Split(fmt.Sprintf("docker network rm %s", se.config.DockerPrefix), " ")
	r, err = exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		log.Warn(string(r))
	}
}

func (se *Stateth) importGrafanaDatasource() error {
	log.Info("Importing Grafana datasource")

	dataSource := &gapi.DataSource{
		Name:      "metrics",
		Type:      "influxdb",
		URL:       fmt.Sprintf("http://%s_influxdb:%d", se.config.DockerPrefix, se.config.InfluxDBPort),
		Access:    "proxy",
		Database:  "metrics",
		User:      influxDBAdminUser,
		Password:  influxDBAdminPass,
		IsDefault: true,
		BasicAuth: false,
	}

	_, err := se.gclient.NewDataSource(dataSource)
	if err != nil {
		log.Warn(err.Error())
		return err
	}

	return nil
}

func (se *Stateth) importGrafanaDashboards() error {
	log.Info("Importing Grafana dashboards")

	files, err := ioutil.ReadDir(se.config.DashboardsFolder)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	for _, f := range files {
		name := f.Name()
		if strings.Contains(name, "json") {
			log.Info("Importing dashboard", "dashboard", name)

			blob, err := ioutil.ReadFile(filepath.Join(se.config.DashboardsFolder, name))
			if err != nil {
				log.Warn(err.Error())
				return nil
			}

			model := se.prepareDashboardModel(string(blob))

			_, err = se.gclient.SaveDashboard(model, false)
			if err != nil {
				log.Warn(err.Error())
				return nil
			}
		}
	}

	return nil
}

func (se *Stateth) prepareDashboardModel(configJSON string) map[string]interface{} {
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		panic("Invalid JSON got into prepare func")
	}

	delete(configMap, "id")
	// Only exists in 5.0+
	delete(configMap, "uid")
	configMap["version"] = 0

	return configMap
}
