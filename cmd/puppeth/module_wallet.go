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

// walletDockerfile is the Dockerfile required to run a web wallet.
var walletDockerfile = `
FROM ethereum/client-go:latest

RUN \
	apk add --update git python make g++ libnotify nodejs-npm && \
	npm install -g gulp-cli

RUN \
  git clone --depth=1 https://github.com/kvhnuke/etherwallet.git && \
	(cd etherwallet && npm install)
WORKDIR etherwallet

RUN \
	echo '"use strict";'                                                  > app/scripts/nodes.js && \
	echo 'var nodes = function() {}'                                     >> app/scripts/nodes.js && \
	echo 'nodes.customNode = require("./nodeHelpers/customNode");'       >> app/scripts/nodes.js && \
	echo 'nodes.nodeTypes = {'                                           >> app/scripts/nodes.js && \
  echo '	{{.Network}}: "{{.Denom}} ETH",'                             >> app/scripts/nodes.js && \
	echo '	Custom: "CUSTOM ETH"'                                        >> app/scripts/nodes.js && \
	echo '};'                                                            >> app/scripts/nodes.js && \
	echo 'nodes.ensNodeTypes = [];'                                      >> app/scripts/nodes.js && \
	echo 'nodes.customNodeObj = {'                                       >> app/scripts/nodes.js && \
  echo '	"name": "CUS",'                                              >> app/scripts/nodes.js && \
  echo '	"type": nodes.nodeTypes.Custom,'                             >> app/scripts/nodes.js && \
  echo '	"eip155": false,'                                            >> app/scripts/nodes.js && \
  echo '	"chainId": "",'                                              >> app/scripts/nodes.js && \
	echo '	"tokenList": [],'                                            >> app/scripts/nodes.js && \
	echo '	"abiList": [],'                                              >> app/scripts/nodes.js && \
	echo '	"service": "Custom",'                                        >> app/scripts/nodes.js && \
  echo '	"lib": null'                                                 >> app/scripts/nodes.js && \
  echo '}'                                                             >> app/scripts/nodes.js && \
	echo 'nodes.nodeList = {'                                            >> app/scripts/nodes.js && \
  echo '	"eth_mew": {'                                                >> app/scripts/nodes.js && \
  echo '		"name": "{{.Network}}",'                                   >> app/scripts/nodes.js && \
  echo '		"type": nodes.nodeTypes.{{.Network}},'                     >> app/scripts/nodes.js && \
  echo '		"eip155": true,'                                           >> app/scripts/nodes.js && \
  echo '		"chainId": {{.NetworkID}},'                                >> app/scripts/nodes.js && \
	echo '		"tokenList": [],'                                          >> app/scripts/nodes.js && \
	echo '		"abiList": [],'                                            >> app/scripts/nodes.js && \
	echo '		"service": "Go Ethereum",'                                 >> app/scripts/nodes.js && \
  echo '		"lib": new nodes.customNode("http://{{.Host}}:{{.RPCPort}}", "")' >> app/scripts/nodes.js && \
  echo '	}'                                                           >> app/scripts/nodes.js && \
	echo '};'                                                            >> app/scripts/nodes.js && \
	echo 'nodes.ethPrice = require("./nodeHelpers/ethPrice");'           >> app/scripts/nodes.js && \
	echo 'module.exports = nodes;'                                       >> app/scripts/nodes.js

RUN rm -rf dist && gulp prep && npm run dist

RUN \
	npm install connect serve-static && \
	\
	echo 'var connect = require("connect");'                                       > server.js && \
	echo 'var serveStatic = require("serve-static");'                             >> server.js && \
	echo 'connect().use(serveStatic("/etherwallet/dist")).listen(80, function(){' >> server.js && \
	echo '    console.log("Server running on 80...");'                            >> server.js && \
	echo '});'                                                                    >> server.js

ADD genesis.json /genesis.json

RUN \
  echo 'node server.js &'                     > wallet.sh && \
	echo 'geth --cache 512 init /genesis.json' >> wallet.sh && \
	echo $'geth --networkid {{.NetworkID}} --port {{.NodePort}} --bootnodes {{.Bootnodes}} --ethstats \'{{.Ethstats}}\' --cache=512 --rpc --rpcaddr=0.0.0.0 --rpccorsdomain "*"' >> wallet.sh

EXPOSE 80 8545

ENTRYPOINT ["/bin/sh", "wallet.sh"]
`

// walletComposefile is the docker-compose.yml file required to deploy and
// maintain a web wallet.
var walletComposefile = `
version: '2'
services:
  wallet:
    build: .
    image: {{.Network}}/wallet
    ports:
      - "{{.NodePort}}:{{.NodePort}}"
      - "{{.NodePort}}:{{.NodePort}}/udp"
      - "{{.RPCPort}}:8545"{{if not .VHost}}
      - "{{.WebPort}}:80"{{end}}
    volumes:
      - {{.Datadir}}:/root/.ethereum
    environment:
      - NODE_PORT={{.NodePort}}/tcp
      - STATS={{.Ethstats}}{{if .VHost}}
      - VIRTUAL_HOST={{.VHost}}
      - VIRTUAL_PORT=80{{end}}
    logging:
      driver: "json-file"
      options:
        max-size: "1m"
        max-file: "10"
    restart: always
`

// deployWallet deploys a new web wallet container to a remote machine via SSH,
// docker and docker-compose. If an instance with the specified network name
// already exists there, it will be overwritten!
func deployWallet(client *sshClient, network string, bootnodes []string, config *walletInfos, nocache bool) ([]byte, error) {
	// Generate the content to upload to the server
	workdir := fmt.Sprintf("%d", rand.Int63())
	files := make(map[string][]byte)

	dockerfile := new(bytes.Buffer)
	template.Must(template.New("").Parse(walletDockerfile)).Execute(dockerfile, map[string]interface{}{
		"Network":   strings.ToTitle(network),
		"Denom":     strings.ToUpper(network),
		"NetworkID": config.network,
		"NodePort":  config.nodePort,
		"RPCPort":   config.rpcPort,
		"Bootnodes": strings.Join(bootnodes, ","),
		"Ethstats":  config.ethstats,
		"Host":      client.address,
	})
	files[filepath.Join(workdir, "Dockerfile")] = dockerfile.Bytes()

	composefile := new(bytes.Buffer)
	template.Must(template.New("").Parse(walletComposefile)).Execute(composefile, map[string]interface{}{
		"Datadir":  config.datadir,
		"Network":  network,
		"NodePort": config.nodePort,
		"RPCPort":  config.rpcPort,
		"VHost":    config.webHost,
		"WebPort":  config.webPort,
		"Ethstats": config.ethstats[:strings.Index(config.ethstats, ":")],
	})
	files[filepath.Join(workdir, "docker-compose.yaml")] = composefile.Bytes()

	files[filepath.Join(workdir, "genesis.json")] = []byte(config.genesis)

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

// walletInfos is returned from a web wallet status check to allow reporting
// various configuration parameters.
type walletInfos struct {
	genesis  []byte
	network  int64
	datadir  string
	ethstats string
	nodePort int
	rpcPort  int
	webHost  string
	webPort  int
}

// Report converts the typed struct into a plain string->string map, cotnaining
// most - but not all - fields for reporting to the user.
func (info *walletInfos) Report() map[string]string {
	report := map[string]string{
		"Data directory":         info.datadir,
		"Ethstats username":      info.ethstats,
		"Node listener port ":    strconv.Itoa(info.nodePort),
		"RPC listener port ":     strconv.Itoa(info.rpcPort),
		"Website address ":       info.webHost,
		"Website listener port ": strconv.Itoa(info.webPort),
	}
	return report
}

// checkWallet does a health-check against web wallet server to verify whether
// it's running, and if yes, whether it's responsive.
func checkWallet(client *sshClient, network string) (*walletInfos, error) {
	// Inspect a possible web wallet container on the host
	infos, err := inspectContainer(client, fmt.Sprintf("%s_wallet_1", network))
	if err != nil {
		return nil, err
	}
	if !infos.running {
		return nil, ErrServiceOffline
	}
	// Resolve the port from the host, or the reverse proxy
	webPort := infos.portmap["80/tcp"]
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
	// Run a sanity check to see if the devp2p and RPC ports are reachable
	nodePort := infos.portmap[infos.envvars["NODE_PORT"]]
	if err = checkPort(client.server, nodePort); err != nil {
		log.Warn(fmt.Sprintf("Wallet devp2p port seems unreachable"), "server", client.server, "port", nodePort, "err", err)
	}
	rpcPort := infos.portmap["8545/tcp"]
	if err = checkPort(client.server, rpcPort); err != nil {
		log.Warn(fmt.Sprintf("Wallet RPC port seems unreachable"), "server", client.server, "port", rpcPort, "err", err)
	}
	// Assemble and return the useful infos
	stats := &walletInfos{
		datadir:  infos.volumes["/root/.ethereum"],
		nodePort: nodePort,
		rpcPort:  rpcPort,
		webHost:  host,
		webPort:  webPort,
		ethstats: infos.envvars["STATS"],
	}
	return stats, nil
}
