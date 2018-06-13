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
	"os"
)

// DefaultConfig contains default settings for the stateth.
var DefaultConfig = Config{
	DockerPrefix:     "stateth",
	DashboardsFolder: os.Getenv("GOPATH") + "/src/github.com/ethereum/go-ethereum/stateth/grafana_dashboards",
	GrafanaPort:      3000,
	InfluxDBPort:     8086,
}

// Config contains the configuration parameters of the dashboard.
type Config struct {
	// DockerPrefix is a unique prefix used for the created docker resources.
	DockerPrefix string `toml:",omitempty"`

	// DashboardsFolder is a folder containing all dashboards to be imported in Grafana.
	DashboardsFolder string `toml:",omitempty"`

	// GrafanaPort is the expose port for the Grafana HTTP interface.
	GrafanaPort int `toml:",omitempty"`

	// InfluxDBPort is the expose port for the InfluxDB HTTP interface.
	InfluxDBPort int `toml:",omitempty"`
}
