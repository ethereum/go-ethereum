// Copyright 2017 The go-ethereum Authors
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

package adapters

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

// DockerAdapter is a NodeAdapter which runs simulation nodes inside Docker
// containers.
//
// A Docker image is built which contains the current binary at /bin/p2p-node
// which when executed runs the underlying service (see the description
// of the execP2PNode function for more details)
type DockerAdapter struct {
	ExecAdapter
}

// NewDockerAdapter builds the p2p-node Docker image containing the current
// binary and returns a DockerAdapter
func NewDockerAdapter() (*DockerAdapter, error) {
	// Since Docker containers run on Linux and this adapter runs the
	// current binary in the container, it must be compiled for Linux.
	//
	// It is reasonable to require this because the caller can just
	// compile the current binary in a Docker container.
	if runtime.GOOS != "linux" {
		return nil, errors.New("DockerAdapter can only be used on Linux as it uses the current binary (which must be a Linux binary)")
	}

	if err := buildDockerImage(); err != nil {
		return nil, err
	}

	return &DockerAdapter{
		ExecAdapter{
			nodes: make(map[discover.NodeID]*ExecNode),
		},
	}, nil
}

// Name returns the name of the adapter for logging purposes
func (d *DockerAdapter) Name() string {
	return "docker-adapter"
}

// NewNode returns a new DockerNode using the given config
func (d *DockerAdapter) NewNode(config *NodeConfig) (Node, error) {
	if len(config.Services) == 0 {
		return nil, errors.New("node must have at least one service")
	}
	for _, service := range config.Services {
		if _, exists := serviceFuncs[service]; !exists {
			return nil, fmt.Errorf("unknown node service %q", service)
		}
	}

	// generate the config
	conf := &execNodeConfig{
		Stack: node.DefaultConfig,
		Node:  config,
	}
	conf.Stack.DataDir = "/data"
	conf.Stack.WSHost = "0.0.0.0"
	conf.Stack.WSOrigins = []string{"*"}
	conf.Stack.WSExposeAll = true
	conf.Stack.P2P.EnableMsgEvents = false
	conf.Stack.P2P.NoDiscovery = true
	conf.Stack.P2P.NAT = nil
	conf.Stack.NoUSB = true

	node := &DockerNode{
		ExecNode: ExecNode{
			ID:      config.ID,
			Config:  conf,
			adapter: &d.ExecAdapter,
		},
	}
	node.newCmd = node.dockerCommand
	d.ExecAdapter.nodes[node.ID] = &node.ExecNode
	return node, nil
}

// DockerNode wraps an ExecNode but exec's the current binary in a docker
// container rather than locally
type DockerNode struct {
	ExecNode
}

// dockerCommand returns a command which exec's the binary in a Docker
// container.
//
// It uses a shell so that we can pass the _P2P_NODE_CONFIG environment
// variable to the container using the --env flag.
func (n *DockerNode) dockerCommand() *exec.Cmd {
	return exec.Command(
		"sh", "-c",
		fmt.Sprintf(
			`exec docker run --interactive --env _P2P_NODE_CONFIG="${_P2P_NODE_CONFIG}" %s p2p-node %s %s`,
			dockerImage, strings.Join(n.Config.Node.Services, ","), n.ID.String(),
		),
	)
}

// dockerImage is the name of the Docker image which gets built to run the
// simulation node
const dockerImage = "p2p-node"

// buildDockerImage builds the Docker image which is used to run the simulation
// node in a Docker container.
//
// It adds the current binary as "p2p-node" so that it runs execP2PNode
// when executed.
func buildDockerImage() error {
	// create a directory to use as the build context
	dir, err := ioutil.TempDir("", "p2p-docker")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	// copy the current binary into the build context
	bin, err := os.Open(reexec.Self())
	if err != nil {
		return err
	}
	defer bin.Close()
	dst, err := os.OpenFile(filepath.Join(dir, "self.bin"), os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, bin); err != nil {
		return err
	}

	// create the Dockerfile
	dockerfile := []byte(`
FROM ubuntu:16.04
RUN mkdir /data
ADD self.bin /bin/p2p-node
	`)
	if err := ioutil.WriteFile(filepath.Join(dir, "Dockerfile"), dockerfile, 0644); err != nil {
		return err
	}

	// run 'docker build'
	cmd := exec.Command("docker", "build", "-t", dockerImage, dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error building docker image: %s", err)
	}

	return nil
}
