// Copyright 2017 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

// explorerDockerfile is the Dockerfile required to run a block explorer.
var explorerDockerfile = `
FROM puppeth/explorer:latest

ADD ethstats.json /ethstats.json
ADD chain.json /chain.json

RUN \
  echo '(cd ../eth-net-intelligence-api && pm2 start /ethstats.json)' >  explorer.sh && \
	echo '(cd ../etherchain-light && npm start &)'                      >> explorer.sh && \
	echo '/parity/parity --chain=/chain.json --port={{.NodePort}} --tracing=on --fat-db=on --pruning=archive' >> explorer.sh

ENTRYPOINT ["/bin/sh", "explorer.sh"]
`

// explorerEthstats is the configuration file for the ethstats javascript client.
var explorerEthstats = `[
  {
    "name"              : "node-app",
    "script"            : "app.js",
    "log_date_format"   : "YYYY-MM-DD HH:mm Z",
    "merge_logs"        : false,
    "watch"             : false,
    "max_restarts"      : 10,
    "exec_interpreter"  : "node",
    "exec_mode"         : "fork_mode",
    "env":
    {
      "NODE_ENV"        : "production",
      "RPC_HOST"        : "localhost",
      "RPC_PORT"        : "8545",
      "LISTENING_PORT"  : "{{.Port}}",
      "INSTANCE_NAME"   : "{{.Name}}",
      "CONTACT_DETAILS" : "",
      "WS_SERVER"       : "{{.Host}}",
      "WS_SECRET"       : "{{.Secret}}",
      "VERBOSITY"       : 2
    }
  }
]`

// explorerComposefile is the docker-compose.yml file required to deploy and
// maintain a block explorer.
var explorerComposefile = `
version: '2'
services:
  explorer:
    build: .
    image: {{.Network}}/explorer
    ports:
      - "{{.NodePort}}:{{.NodePort}}"
      - "{{.NodePort}}:{{.NodePort}}/udp"{{if not .VHost}}
      - "{{.WebPort}}:3000"{{end}}
    volumes:
      - {{.Datadir}}:/root/.local/share/io.parity.ethereum
    environment:
      - NODE_PORT={{.NodePort}}/tcp
      - STATS={{.Ethstats}}{{if .VHost}}
      - VIRTUAL_HOST={{.VHost}}
      - VIRTUAL_PORT=3000{{end}}
    logging:
      driver: "json-file"
      options:
        max-size: "1m"
        max-file: "10"
    restart: always
`

// deployExplorer deploys a new block explorer container to a remote machine via
// SSH, docker and docker-compose. If an instance with the specified network name
// already exists there, it will be overwritten!
func deployExplorer(client *sshClient, network string, chainspec []byte, config *explorerInfos, nocache bool) ([]byte, error) {
	// Generate the content to upload to the server
	workdir := fmt.Sprintf("%d", rand.Int63())
	files := make(map[string][]byte)

	dockerfile := new(bytes.Buffer)
	template.Must(template.New("").Parse(explorerDockerfile)).Execute(dockerfile, map[string]interface{}{
		"NodePort": config.nodePort,
	})
	files[filepath.Join(workdir, "Dockerfile")] = dockerfile.Bytes()

	ethstats := new(bytes.Buffer)
	template.Must(template.New("").Parse(explorerEthstats)).Execute(ethstats, map[string]interface{}{
		"Port":   config.nodePort,
		"Name":   config.ethstats[:strings.Index(config.ethstats, ":")],
		"Secret": config.ethstats[strings.Index(config.ethstats, ":")+1 : strings.Index(config.ethstats, "@")],
		"Host":   config.ethstats[strings.Index(config.ethstats, "@")+1:],
	})
	files[filepath.Join(workdir, "ethstats.json")] = ethstats.Bytes()

	composefile := new(bytes.Buffer)
	template.Must(template.New("").Parse(explorerComposefile)).Execute(composefile, map[string]interface{}{
		"Datadir":  config.datadir,
		"Network":  network,
		"NodePort": config.nodePort,
		"VHost":    config.webHost,
		"WebPort":  config.webPort,
		"Ethstats": config.ethstats[:strings.Index(config.ethstats, ":")],
	})
	files[filepath.Join(workdir, "docker-compose.yaml")] = composefile.Bytes()

	files[filepath.Join(workdir, "chain.json")] = chainspec

	// Upload the deployment files to the remote server (and clean up afterwards)
	if out, err := client.Upload(files); err != nil {
		return out, err
	}
	defer client.Run("rm -rf " + workdir)

	// Build and deploy the boot or seal node service
	if nocache {
		return nil, client.Stream(fmt.Sprintf("cd %s && docker-compose -p %s build --pull --no-cache && docker-compose -p %s up -d --force-recreate", workdir, network, network))
	}
	return nil, client.Stream(fmt.Sprintf("cd %s && docker-compose -p %s up -d --build --force-recreate", workdir, network))
}

// explorerInfos is returned from a block explorer status check to allow reporting
// various configuration parameters.
type explorerInfos struct {
	datadir  string
	ethstats string
	nodePort int
	webHost  string
	webPort  int
}

// Report converts the typed struct into a plain string->string map, containing
// most - but not all - fields for reporting to the user.
func (info *explorerInfos) Report() map[string]string {
	report := map[string]string{
		"Data directory":         info.datadir,
		"Node listener port ":    strconv.Itoa(info.nodePort),
		"Ethstats username":      info.ethstats,
		"Website address ":       info.webHost,
		"Website listener port ": strconv.Itoa(info.webPort),
	}
	return report
}

// checkExplorer does a health-check against a block explorer server to verify
// whether it's running, and if yes, whether it's responsive.
func checkExplorer(client *sshClient, network string) (*explorerInfos, error) {
	// Inspect a possible block explorer container on the host
	infos, err := inspectContainer(client, fmt.Sprintf("%s_explorer_1", network))
	if err != nil {
		return nil, err
	}
	if !infos.running {
		return nil, ErrServiceOffline
	}
	// Resolve the port from the host, or the reverse proxy
	webPort := infos.portmap["3000/tcp"]
	if webPort == 0 {
		if proxy, _ := checkNginx(client, network); proxy != nil {
			webPort = proxy.port
		}
	}
	if webPort == 0 {
		return nil, ErrNotExposed
	}
	// Resolve the host from the reverse-proxy and the config values
	host := infos.envvars["VIRTUAL_HOST"]
	if host == "" {
		host = client.server
	}
	// Run a sanity check to see if the devp2p is reachable
	nodePort := infos.portmap[infos.envvars["NODE_PORT"]]
	if err = checkPort(client.server, nodePort); err != nil {
		log.Warn(fmt.Sprintf("Explorer devp2p port seems unreachable"), "server", client.server, "port", nodePort, "err", err)
	}
	// Assemble and return the useful infos
	stats := &explorerInfos{
		datadir:  infos.volumes["/root/.local/share/io.parity.ethereum"],
		nodePort: nodePort,
		webHost:  host,
		webPort:  webPort,
		ethstats: infos.envvars["STATS"],
	}
	return stats, nil
}
