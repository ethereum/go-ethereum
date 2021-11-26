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
FROM puppeth/blockscout:latest

ADD genesis.json /genesis.json
RUN \
  echo 'geth --cache 512 init /genesis.json' > explorer.sh && \
  echo $'geth --networkid {{.NetworkID}} --syncmode "full" --gcmode "archive" --port {{.EthPort}} --bootnodes {{.Bootnodes}} --ethstats \'{{.Ethstats}}\' --cache=512 --http --http.api "net,web3,eth,debug,txpool" --http.corsdomain "*" --http.vhosts "*" --ws --ws.origins "*" --exitwhensynced' >> explorer.sh && \
  echo $'exec geth --networkid {{.NetworkID}} --syncmode "full" --gcmode "archive" --port {{.EthPort}} --bootnodes {{.Bootnodes}} --ethstats \'{{.Ethstats}}\' --cache=512 --http --http.api "net,web3,eth,debug,txpool" --http.corsdomain "*" --http.vhosts "*" --ws --ws.origins "*" &' >> explorer.sh && \
  echo '/usr/local/bin/docker-entrypoint.sh postgres &' >> explorer.sh && \
  echo 'sleep 5' >> explorer.sh && \
  echo 'mix do ecto.drop --force, ecto.create, ecto.migrate' >> explorer.sh && \
  echo 'mix phx.server' >> explorer.sh

ENTRYPOINT ["/bin/sh", "explorer.sh"]
`

// explorerComposefile is the docker-compose.yml file required to deploy and
// maintain a block explorer.
var explorerComposefile = `
version: '2'
services:
    explorer:
        build: .
        image: {{.Network}}/explorer
        container_name: {{.Network}}_explorer_1
        ports:
            - "{{.EthPort}}:{{.EthPort}}"
            - "{{.EthPort}}:{{.EthPort}}/udp"{{if not .VHost}}
            - "{{.WebPort}}:4000"{{end}}
        environment:
            - ETH_PORT={{.EthPort}}
            - ETH_NAME={{.EthName}}
            - BLOCK_TRANSFORMER={{.Transformer}}{{if .VHost}}
            - VIRTUAL_HOST={{.VHost}}
            - VIRTUAL_PORT=4000{{end}}
        volumes:
            - {{.Datadir}}:/opt/app/.ethereum
            - {{.DBDir}}:/var/lib/postgresql/data
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
func deployExplorer(client *sshClient, network string, bootnodes []string, config *explorerInfos, nocache bool, isClique bool) ([]byte, error) {
	// Generate the content to upload to the server
	workdir := fmt.Sprintf("%d", rand.Int63())
	files := make(map[string][]byte)

	dockerfile := new(bytes.Buffer)
	template.Must(template.New("").Parse(explorerDockerfile)).Execute(dockerfile, map[string]interface{}{
		"NetworkID": config.node.network,
		"Bootnodes": strings.Join(bootnodes, ","),
		"Ethstats":  config.node.ethstats,
		"EthPort":   config.node.port,
	})
	files[filepath.Join(workdir, "Dockerfile")] = dockerfile.Bytes()

	transformer := "base"
	if isClique {
		transformer = "clique"
	}
	composefile := new(bytes.Buffer)
	template.Must(template.New("").Parse(explorerComposefile)).Execute(composefile, map[string]interface{}{
		"Network":     network,
		"VHost":       config.host,
		"Ethstats":    config.node.ethstats,
		"Datadir":     config.node.datadir,
		"DBDir":       config.dbdir,
		"EthPort":     config.node.port,
		"EthName":     config.node.ethstats[:strings.Index(config.node.ethstats, ":")],
		"WebPort":     config.port,
		"Transformer": transformer,
	})
	files[filepath.Join(workdir, "docker-compose.yaml")] = composefile.Bytes()
	files[filepath.Join(workdir, "genesis.json")] = config.node.genesis

	// Upload the deployment files to the remote server (and clean up afterwards)
	if out, err := client.Upload(files); err != nil {
		return out, err
	}
	defer client.Run("rm -rf " + workdir)

	// Build and deploy the boot or seal node service
	if nocache {
		return nil, client.Stream(fmt.Sprintf("cd %s && docker-compose -p %s build --pull --no-cache && docker-compose -p %s up -d --force-recreate --timeout 60", workdir, network, network))
	}
	return nil, client.Stream(fmt.Sprintf("cd %s && docker-compose -p %s up -d --build --force-recreate --timeout 60", workdir, network))
}

// explorerInfos is returned from a block explorer status check to allow reporting
// various configuration parameters.
type explorerInfos struct {
	node  *nodeInfos
	dbdir string
	host  string
	port  int
}

// Report converts the typed struct into a plain string->string map, containing
// most - but not all - fields for reporting to the user.
func (info *explorerInfos) Report() map[string]string {
	report := map[string]string{
		"Website address ":        info.host,
		"Website listener port ":  strconv.Itoa(info.port),
		"Ethereum listener port ": strconv.Itoa(info.node.port),
		"Ethstats username":       info.node.ethstats,
	}
	return report
}

// checkExplorer does a health-check against a block explorer server to verify
// whether it's running, and if yes, whether it's responsive.
func checkExplorer(client *sshClient, network string) (*explorerInfos, error) {
	// Inspect a possible explorer container on the host
	infos, err := inspectContainer(client, fmt.Sprintf("%s_explorer_1", network))
	if err != nil {
		return nil, err
	}
	if !infos.running {
		return nil, ErrServiceOffline
	}
	// Resolve the port from the host, or the reverse proxy
	port := infos.portmap["4000/tcp"]
	if port == 0 {
		if proxy, _ := checkNginx(client, network); proxy != nil {
			port = proxy.port
		}
	}
	if port == 0 {
		return nil, ErrNotExposed
	}
	// Resolve the host from the reverse-proxy and the config values
	host := infos.envvars["VIRTUAL_HOST"]
	if host == "" {
		host = client.server
	}
	// Run a sanity check to see if the devp2p is reachable
	p2pPort := infos.portmap[infos.envvars["ETH_PORT"]+"/tcp"]
	if err = checkPort(host, p2pPort); err != nil {
		log.Warn("Explorer node seems unreachable", "server", host, "port", p2pPort, "err", err)
	}
	if err = checkPort(host, port); err != nil {
		log.Warn("Explorer service seems unreachable", "server", host, "port", port, "err", err)
	}
	// Assemble and return the useful infos
	stats := &explorerInfos{
		node: &nodeInfos{
			datadir:  infos.volumes["/opt/app/.ethereum"],
			port:     infos.portmap[infos.envvars["ETH_PORT"]+"/tcp"],
			ethstats: infos.envvars["ETH_NAME"],
		},
		dbdir: infos.volumes["/var/lib/postgresql/data"],
		host:  host,
		port:  port,
	}
	return stats, nil
}
