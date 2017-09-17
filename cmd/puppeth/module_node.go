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
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/ethereum/go-ethereum/log"
)

// nodeDockerfile is the Dockerfile required to run an Ethereum node.
var nodeDockerfile = `
FROM ethereum/client-go:latest

ADD genesis.json /genesis.json
{{if .Unlock}}
	ADD signer.json /signer.json
	ADD signer.pass /signer.pass
{{end}}
RUN \
  echo 'geth init /genesis.json' > geth.sh && \{{if .Unlock}}
	echo 'mkdir -p /root/.ethereum/keystore/ && cp /signer.json /root/.ethereum/keystore/' >> geth.sh && \{{end}}
	echo $'geth --networkid {{.NetworkID}} --cache 512 --port {{.Port}} --maxpeers {{.Peers}} {{.LightFlag}} --ethstats \'{{.Ethstats}}\' {{if .BootV4}}--bootnodesv4 {{.BootV4}}{{end}} {{if .BootV5}}--bootnodesv5 {{.BootV5}}{{end}} {{if .Etherbase}}--etherbase {{.Etherbase}} --mine{{end}}{{if .Unlock}}--unlock 0 --password /signer.pass --mine{{end}} --targetgaslimit {{.GasTarget}} --gasprice {{.GasPrice}}' >> geth.sh

ENTRYPOINT ["/bin/sh", "geth.sh"]
`

// nodeComposefile is the docker-compose.yml file required to deploy and maintain
// an Ethereum node (bootnode or miner for now).
var nodeComposefile = `
version: '2'
services:
  {{.Type}}:
    build: .
    image: {{.Network}}/{{.Type}}
    ports:
      - "{{.FullPort}}:{{.FullPort}}"
      - "{{.FullPort}}:{{.FullPort}}/udp"{{if .Light}}
      - "{{.LightPort}}:{{.LightPort}}/udp"{{end}}
    volumes:
      - {{.Datadir}}:/root/.ethereum
    environment:
      - FULL_PORT={{.FullPort}}/tcp
      - LIGHT_PORT={{.LightPort}}/udp
      - TOTAL_PEERS={{.TotalPeers}}
      - LIGHT_PEERS={{.LightPeers}}
      - STATS_NAME={{.Ethstats}}
      - MINER_NAME={{.Etherbase}}
      - GAS_TARGET={{.GasTarget}}
      - GAS_PRICE={{.GasPrice}}
    logging:
      driver: "json-file"
      options:
        max-size: "1m"
        max-file: "10"
    restart: always
`

// deployNode deploys a new Ethereum node container to a remote machine via SSH,
// docker and docker-compose. If an instance with the specified network name
// already exists there, it will be overwritten!
func deployNode(client *sshClient, network string, bootv4, bootv5 []string, config *nodeInfos) ([]byte, error) {
	kind := "sealnode"
	if config.keyJSON == "" && config.etherbase == "" {
		kind = "bootnode"
		bootv4 = make([]string, 0)
		bootv5 = make([]string, 0)
	}
	// Generate the content to upload to the server
	workdir := fmt.Sprintf("%d", rand.Int63())
	files := make(map[string][]byte)

	lightFlag := ""
	if config.peersLight > 0 {
		lightFlag = fmt.Sprintf("--lightpeers=%d --lightserv=50", config.peersLight)
	}
	dockerfile := new(bytes.Buffer)
	template.Must(template.New("").Parse(nodeDockerfile)).Execute(dockerfile, map[string]interface{}{
		"NetworkID": config.network,
		"Port":      config.portFull,
		"Peers":     config.peersTotal,
		"LightFlag": lightFlag,
		"BootV4":    strings.Join(bootv4, ","),
		"BootV5":    strings.Join(bootv5, ","),
		"Ethstats":  config.ethstats,
		"Etherbase": config.etherbase,
		"GasTarget": uint64(1000000 * config.gasTarget),
		"GasPrice":  uint64(1000000000 * config.gasPrice),
		"Unlock":    config.keyJSON != "",
	})
	files[filepath.Join(workdir, "Dockerfile")] = dockerfile.Bytes()

	composefile := new(bytes.Buffer)
	template.Must(template.New("").Parse(nodeComposefile)).Execute(composefile, map[string]interface{}{
		"Type":       kind,
		"Datadir":    config.datadir,
		"Network":    network,
		"FullPort":   config.portFull,
		"TotalPeers": config.peersTotal,
		"Light":      config.peersLight > 0,
		"LightPort":  config.portFull + 1,
		"LightPeers": config.peersLight,
		"Ethstats":   config.ethstats[:strings.Index(config.ethstats, ":")],
		"Etherbase":  config.etherbase,
		"GasTarget":  config.gasTarget,
		"GasPrice":   config.gasPrice,
	})
	files[filepath.Join(workdir, "docker-compose.yaml")] = composefile.Bytes()

	//genesisfile, _ := json.MarshalIndent(config.genesis, "", "  ")
	files[filepath.Join(workdir, "genesis.json")] = []byte(config.genesis)

	if config.keyJSON != "" {
		files[filepath.Join(workdir, "signer.json")] = []byte(config.keyJSON)
		files[filepath.Join(workdir, "signer.pass")] = []byte(config.keyPass)
	}
	// Upload the deployment files to the remote server (and clean up afterwards)
	if out, err := client.Upload(files); err != nil {
		return out, err
	}
	defer client.Run("rm -rf " + workdir)

	// Build and deploy the boot or seal node service
	return nil, client.Stream(fmt.Sprintf("cd %s && docker-compose -p %s up -d --build", workdir, network))
}

// nodeInfos is returned from a boot or seal node status check to allow reporting
// various configuration parameters.
type nodeInfos struct {
	genesis    []byte
	network    int64
	datadir    string
	ethstats   string
	portFull   int
	portLight  int
	enodeFull  string
	enodeLight string
	peersTotal int
	peersLight int
	etherbase  string
	keyJSON    string
	keyPass    string
	gasTarget  float64
	gasPrice   float64
}

// String implements the stringer interface.
func (info *nodeInfos) String() string {
	discv5 := ""
	if info.peersLight > 0 {
		discv5 = fmt.Sprintf(", portv5=%d", info.portLight)
	}
	return fmt.Sprintf("port=%d%s, datadir=%s, peers=%d, lights=%d, ethstats=%s, gastarget=%0.3f MGas, gasprice=%0.3f GWei",
		info.portFull, discv5, info.datadir, info.peersTotal, info.peersLight, info.ethstats, info.gasTarget, info.gasPrice)
}

// checkNode does a health-check against an boot or seal node server to verify
// whether it's running, and if yes, whether it's responsive.
func checkNode(client *sshClient, network string, boot bool) (*nodeInfos, error) {
	kind := "bootnode"
	if !boot {
		kind = "sealnode"
	}
	// Inspect a possible bootnode container on the host
	infos, err := inspectContainer(client, fmt.Sprintf("%s_%s_1", network, kind))
	if err != nil {
		return nil, err
	}
	if !infos.running {
		return nil, ErrServiceOffline
	}
	// Resolve a few types from the environmental variables
	totalPeers, _ := strconv.Atoi(infos.envvars["TOTAL_PEERS"])
	lightPeers, _ := strconv.Atoi(infos.envvars["LIGHT_PEERS"])
	gasTarget, _ := strconv.ParseFloat(infos.envvars["GAS_TARGET"], 64)
	gasPrice, _ := strconv.ParseFloat(infos.envvars["GAS_PRICE"], 64)

	// Container available, retrieve its node ID and its genesis json
	var out []byte
	if out, err = client.Run(fmt.Sprintf("docker exec %s_%s_1 geth --exec admin.nodeInfo.id attach", network, kind)); err != nil {
		return nil, ErrServiceUnreachable
	}
	id := bytes.Trim(bytes.TrimSpace(out), "\"")

	if out, err = client.Run(fmt.Sprintf("docker exec %s_%s_1 cat /genesis.json", network, kind)); err != nil {
		return nil, ErrServiceUnreachable
	}
	genesis := bytes.TrimSpace(out)

	keyJSON, keyPass := "", ""
	if out, err = client.Run(fmt.Sprintf("docker exec %s_%s_1 cat /signer.json", network, kind)); err == nil {
		keyJSON = string(bytes.TrimSpace(out))
	}
	if out, err = client.Run(fmt.Sprintf("docker exec %s_%s_1 cat /signer.pass", network, kind)); err == nil {
		keyPass = string(bytes.TrimSpace(out))
	}
	// Run a sanity check to see if the devp2p is reachable
	port := infos.portmap[infos.envvars["FULL_PORT"]]
	if err = checkPort(client.server, port); err != nil {
		log.Warn(fmt.Sprintf("%s devp2p port seems unreachable", strings.Title(kind)), "server", client.server, "port", port, "err", err)
	}
	// Assemble and return the useful infos
	stats := &nodeInfos{
		genesis:    genesis,
		datadir:    infos.volumes["/root/.ethereum"],
		portFull:   infos.portmap[infos.envvars["FULL_PORT"]],
		portLight:  infos.portmap[infos.envvars["LIGHT_PORT"]],
		peersTotal: totalPeers,
		peersLight: lightPeers,
		ethstats:   infos.envvars["STATS_NAME"],
		etherbase:  infos.envvars["MINER_NAME"],
		keyJSON:    keyJSON,
		keyPass:    keyPass,
		gasTarget:  gasTarget,
		gasPrice:   gasPrice,
	}
	stats.enodeFull = fmt.Sprintf("enode://%s@%s:%d", id, client.address, stats.portFull)
	if stats.portLight != 0 {
		stats.enodeLight = fmt.Sprintf("enode://%s@%s:%d?discport=%d", id, client.address, stats.portFull, stats.portLight)
	}
	return stats, nil
}
