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
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// faucetDockerfile is the Dockerfile required to build a faucet container to
// grant crypto tokens based on GitHub authentications.
var faucetDockerfile = `
FROM ethereum/client-go:alltools-latest

ADD genesis.json /genesis.json
ADD account.json /account.json
ADD account.pass /account.pass

EXPOSE 8080 30303 30303/udp

ENTRYPOINT [ \
	"faucet", "--genesis", "/genesis.json", "--network", "{{.NetworkID}}", "--bootnodes", "{{.Bootnodes}}", "--ethstats", "{{.Ethstats}}", "--ethport", "{{.EthPort}}",     \
	"--faucet.name", "{{.FaucetName}}", "--faucet.amount", "{{.FaucetAmount}}", "--faucet.minutes", "{{.FaucetMinutes}}", "--faucet.tiers", "{{.FaucetTiers}}",             \
	"--account.json", "/account.json", "--account.pass", "/account.pass"                                                                                                    \
	{{if .CaptchaToken}}, "--captcha.token", "{{.CaptchaToken}}", "--captcha.secret", "{{.CaptchaSecret}}"{{end}}{{if .NoAuth}}, "--noauth"{{end}}                          \
]`

// faucetComposefile is the docker-compose.yml file required to deploy and maintain
// a crypto faucet.
var faucetComposefile = `
version: '2'
services:
  faucet:
    build: .
    image: {{.Network}}/faucet
    ports:
      - "{{.EthPort}}:{{.EthPort}}"{{if not .VHost}}
      - "{{.ApiPort}}:8080"{{end}}
    volumes:
      - {{.Datadir}}:/root/.faucet
    environment:
      - ETH_PORT={{.EthPort}}
      - ETH_NAME={{.EthName}}
      - FAUCET_AMOUNT={{.FaucetAmount}}
      - FAUCET_MINUTES={{.FaucetMinutes}}
      - FAUCET_TIERS={{.FaucetTiers}}
      - CAPTCHA_TOKEN={{.CaptchaToken}}
      - CAPTCHA_SECRET={{.CaptchaSecret}}
      - NO_AUTH={{.NoAuth}}{{if .VHost}}
      - VIRTUAL_HOST={{.VHost}}
      - VIRTUAL_PORT=8080{{end}}
    logging:
      driver: "json-file"
      options:
        max-size: "1m"
        max-file: "10"
    restart: always
`

// deployFaucet deploys a new faucet container to a remote machine via SSH,
// docker and docker-compose. If an instance with the specified network name
// already exists there, it will be overwritten!
func deployFaucet(client *sshClient, network string, bootnodes []string, config *faucetInfos, nocache bool) ([]byte, error) {
	// Generate the content to upload to the server
	workdir := fmt.Sprintf("%d", rand.Int63())
	files := make(map[string][]byte)

	dockerfile := new(bytes.Buffer)
	template.Must(template.New("").Parse(faucetDockerfile)).Execute(dockerfile, map[string]interface{}{
		"NetworkID":     config.node.network,
		"Bootnodes":     strings.Join(bootnodes, ","),
		"Ethstats":      config.node.ethstats,
		"EthPort":       config.node.port,
		"CaptchaToken":  config.captchaToken,
		"CaptchaSecret": config.captchaSecret,
		"FaucetName":    strings.Title(network),
		"FaucetAmount":  config.amount,
		"FaucetMinutes": config.minutes,
		"FaucetTiers":   config.tiers,
		"NoAuth":        config.noauth,
	})
	files[filepath.Join(workdir, "Dockerfile")] = dockerfile.Bytes()

	composefile := new(bytes.Buffer)
	template.Must(template.New("").Parse(faucetComposefile)).Execute(composefile, map[string]interface{}{
		"Network":       network,
		"Datadir":       config.node.datadir,
		"VHost":         config.host,
		"ApiPort":       config.port,
		"EthPort":       config.node.port,
		"EthName":       config.node.ethstats[:strings.Index(config.node.ethstats, ":")],
		"CaptchaToken":  config.captchaToken,
		"CaptchaSecret": config.captchaSecret,
		"FaucetAmount":  config.amount,
		"FaucetMinutes": config.minutes,
		"FaucetTiers":   config.tiers,
		"NoAuth":        config.noauth,
	})
	files[filepath.Join(workdir, "docker-compose.yaml")] = composefile.Bytes()

	files[filepath.Join(workdir, "genesis.json")] = config.node.genesis
	files[filepath.Join(workdir, "account.json")] = []byte(config.node.keyJSON)
	files[filepath.Join(workdir, "account.pass")] = []byte(config.node.keyPass)

	// Upload the deployment files to the remote server (and clean up afterwards)
	if out, err := client.Upload(files); err != nil {
		return out, err
	}
	defer client.Run("rm -rf " + workdir)

	// Build and deploy the faucet service
	if nocache {
		return nil, client.Stream(fmt.Sprintf("cd %s && docker-compose -p %s build --pull --no-cache && docker-compose -p %s up -d --force-recreate --timeout 60", workdir, network, network))
	}
	return nil, client.Stream(fmt.Sprintf("cd %s && docker-compose -p %s up -d --build --force-recreate --timeout 60", workdir, network))
}

// faucetInfos is returned from a faucet status check to allow reporting various
// configuration parameters.
type faucetInfos struct {
	node          *nodeInfos
	host          string
	port          int
	amount        int
	minutes       int
	tiers         int
	noauth        bool
	captchaToken  string
	captchaSecret string
}

// Report converts the typed struct into a plain string->string map, containing
// most - but not all - fields for reporting to the user.
func (info *faucetInfos) Report() map[string]string {
	report := map[string]string{
		"Website address":              info.host,
		"Website listener port":        strconv.Itoa(info.port),
		"Ethereum listener port":       strconv.Itoa(info.node.port),
		"Funding amount (base tier)":   fmt.Sprintf("%d Ethers", info.amount),
		"Funding cooldown (base tier)": fmt.Sprintf("%d mins", info.minutes),
		"Funding tiers":                strconv.Itoa(info.tiers),
		"Captha protection":            fmt.Sprintf("%v", info.captchaToken != ""),
		"Ethstats username":            info.node.ethstats,
	}
	if info.noauth {
		report["Debug mode (no auth)"] = "enabled"
	}
	if info.node.keyJSON != "" {
		var key struct {
			Address string `json:"address"`
		}
		if err := json.Unmarshal([]byte(info.node.keyJSON), &key); err == nil {
			report["Funding account"] = common.HexToAddress(key.Address).Hex()
		} else {
			log.Error("Failed to retrieve signer address", "err", err)
		}
	}
	return report
}

// checkFaucet does a health-check against a faucet server to verify whether
// it's running, and if yes, gathering a collection of useful infos about it.
func checkFaucet(client *sshClient, network string) (*faucetInfos, error) {
	// Inspect a possible faucet container on the host
	infos, err := inspectContainer(client, fmt.Sprintf("%s_faucet_1", network))
	if err != nil {
		return nil, err
	}
	if !infos.running {
		return nil, ErrServiceOffline
	}
	// Resolve the port from the host, or the reverse proxy
	port := infos.portmap["8080/tcp"]
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
	amount, _ := strconv.Atoi(infos.envvars["FAUCET_AMOUNT"])
	minutes, _ := strconv.Atoi(infos.envvars["FAUCET_MINUTES"])
	tiers, _ := strconv.Atoi(infos.envvars["FAUCET_TIERS"])

	// Retrieve the funding account informations
	var out []byte
	keyJSON, keyPass := "", ""
	if out, err = client.Run(fmt.Sprintf("docker exec %s_faucet_1 cat /account.json", network)); err == nil {
		keyJSON = string(bytes.TrimSpace(out))
	}
	if out, err = client.Run(fmt.Sprintf("docker exec %s_faucet_1 cat /account.pass", network)); err == nil {
		keyPass = string(bytes.TrimSpace(out))
	}
	// Run a sanity check to see if the port is reachable
	if err = checkPort(host, port); err != nil {
		log.Warn("Faucet service seems unreachable", "server", host, "port", port, "err", err)
	}
	// Container available, assemble and return the useful infos
	return &faucetInfos{
		node: &nodeInfos{
			datadir:  infos.volumes["/root/.faucet"],
			port:     infos.portmap[infos.envvars["ETH_PORT"]+"/tcp"],
			ethstats: infos.envvars["ETH_NAME"],
			keyJSON:  keyJSON,
			keyPass:  keyPass,
		},
		host:          host,
		port:          port,
		amount:        amount,
		minutes:       minutes,
		tiers:         tiers,
		captchaToken:  infos.envvars["CAPTCHA_TOKEN"],
		captchaSecret: infos.envvars["CAPTCHA_SECRET"],
		noauth:        infos.envvars["NO_AUTH"] == "true",
	}, nil
}
