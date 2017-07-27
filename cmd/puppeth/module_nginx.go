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

	"github.com/ethereum/go-ethereum/log"
)

// nginxDockerfile is theis the Dockerfile required to build an nginx reverse-
// proxy.
var nginxDockerfile = `FROM jwilder/nginx-proxy`

// nginxComposefile is the docker-compose.yml file required to deploy and maintain
// an nginx reverse-proxy. The proxy is responsible for exposing one or more HTTP
// services running on a single host.
var nginxComposefile = `
version: '2'
services:
  nginx:
    build: .
    image: {{.Network}}/nginx
    ports:
      - "{{.Port}}:80"
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock:ro
    logging:
      driver: "json-file"
      options:
        max-size: "1m"
        max-file: "10"
    restart: always
`

// deployNginx deploys a new nginx reverse-proxy container to expose one or more
// HTTP services running on a single host. If an instance with the specified
// network name already exists there, it will be overwritten!
func deployNginx(client *sshClient, network string, port int) ([]byte, error) {
	log.Info("Deploying nginx reverse-proxy", "server", client.server, "port", port)

	// Generate the content to upload to the server
	workdir := fmt.Sprintf("%d", rand.Int63())
	files := make(map[string][]byte)

	dockerfile := new(bytes.Buffer)
	template.Must(template.New("").Parse(nginxDockerfile)).Execute(dockerfile, nil)
	files[filepath.Join(workdir, "Dockerfile")] = dockerfile.Bytes()

	composefile := new(bytes.Buffer)
	template.Must(template.New("").Parse(nginxComposefile)).Execute(composefile, map[string]interface{}{
		"Network": network,
		"Port":    port,
	})
	files[filepath.Join(workdir, "docker-compose.yaml")] = composefile.Bytes()

	// Upload the deployment files to the remote server (and clean up afterwards)
	if out, err := client.Upload(files); err != nil {
		return out, err
	}
	defer client.Run("rm -rf " + workdir)

	// Build and deploy the ethstats service
	return nil, client.Stream(fmt.Sprintf("cd %s && docker-compose -p %s up -d --build", workdir, network))
}

// nginxInfos is returned from an nginx reverse-proxy status check to allow
// reporting various configuration parameters.
type nginxInfos struct {
	port int
}

// String implements the stringer interface.
func (info *nginxInfos) String() string {
	return fmt.Sprintf("port=%d", info.port)
}

// checkNginx does a health-check against an nginx reverse-proxy to verify whether
// it's running, and if yes, gathering a collection of useful infos about it.
func checkNginx(client *sshClient, network string) (*nginxInfos, error) {
	// Inspect a possible nginx container on the host
	infos, err := inspectContainer(client, fmt.Sprintf("%s_nginx_1", network))
	if err != nil {
		return nil, err
	}
	if !infos.running {
		return nil, ErrServiceOffline
	}
	// Container available, assemble and return the useful infos
	return &nginxInfos{
		port: infos.portmap["80/tcp"],
	}, nil
}
