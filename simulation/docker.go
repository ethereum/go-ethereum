package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethersphere/swarm"
	"github.com/ethersphere/swarm/log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
)

const (
	dockerP2PPort       = 30399
	dockerWebsocketPort = 8546
	dockerHTTPPort      = 8500
	dockerPProfPort     = 6060
)

// DockerAdapter is an adapter that can manage DockerNodes
type DockerAdapter struct {
	client *client.Client
	image  string
	config DockerAdapterConfig
}

// DockerAdapterConfig is the configuration that can be provided when
// initializing a DockerAdapter
type DockerAdapterConfig struct {
	// BuildContext can be used to build a docker image
	// from a Dockerfile and a context directory
	BuildContext *DockerBuildContext `json:"build,omitempty"`
	// DockerImage points to an existing docker image
	// e.g. ethersphere/swarm:latest
	DockerImage string `json:"image,omitempty"`
	// DaemonAddr is the docker daemon address
	DaemonAddr string `json:"daemonAddr,omitempty"`
}

// DockerBuildContext defines the build context to build
// local docker images
type DockerBuildContext struct {
	// Dockefile is the path to the dockerfile
	Dockerfile string `json:"dockerfile"`
	// Directory is the directory that will be used
	// in the context of a docker build
	Directory string `json:"directory"`
	// Tag is used to tag the image
	Tag string `json:"tag"`
}

// DockerNode is a node that was started via the DockerAdapter
type DockerNode struct {
	config  NodeConfig
	adapter *DockerAdapter
	info    NodeInfo
	ipAddr  string
}

// DefaultDockerAdapterConfig returns the default configuration
// that uses the local docker daemon
func DefaultDockerAdapterConfig() DockerAdapterConfig {
	return DockerAdapterConfig{
		DaemonAddr: client.DefaultDockerHost,
	}
}

// DefaultDockerBuildContext returns the default build context that uses a Dockerfile
func DefaultDockerBuildContext() DockerBuildContext {
	return DockerBuildContext{
		Dockerfile: "Dockerfile",
		Directory:  ".",
	}
}

// IsDockerAvailable can be used to check the connectivity to the docker daemon
func IsDockerAvailable(daemonAddr string) bool {
	cli, err := client.NewClientWithOpts(
		client.WithHost(daemonAddr),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return false
	}
	_, err = cli.ServerVersion(context.Background())
	if err != nil {
		return false
	}
	cli.Close()
	return true
}

// NewDockerAdapter creates a DockerAdapter by receiving a DockerAdapterConfig
func NewDockerAdapter(config DockerAdapterConfig) (*DockerAdapter, error) {
	if config.DockerImage != "" && config.BuildContext != nil {
		return nil, fmt.Errorf("only one can be defined: BuildContext (%v) or DockerImage(%s)",
			config.BuildContext, config.DockerImage)
	}

	if config.DockerImage == "" && config.BuildContext == nil {
		return nil, errors.New("required: BuildContext or ExecutablePath")
	}

	// Create docker client
	cli, err := client.NewClientWithOpts(
		client.WithHost(config.DaemonAddr),
		client.WithAPIVersionNegotiation(),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create the docker client: %v", err)
	}

	// Figure out which docker image should be used
	image := config.DockerImage

	// Build docker image
	if config.BuildContext != nil {
		var err error
		image, err = buildImage(*config.BuildContext, config.DaemonAddr)
		if err != nil {
			return nil, fmt.Errorf("could not build the docker image: %v", err)
		}
	}

	// Pull docker image
	if config.DockerImage != "" {
		reader, err := cli.ImagePull(context.Background(), config.DockerImage, types.ImagePullOptions{})
		if err != nil {
			return nil, fmt.Errorf("pull image error: %v", err)
		}
		if _, err := io.Copy(os.Stdout, reader); err != nil && err != io.EOF {
			log.Error("Error pulling docker image", "err", err)
		}
	}

	return &DockerAdapter{
		image:  image,
		client: cli,
		config: config,
	}, nil
}

// NewNode creates a new node
func (a DockerAdapter) NewNode(config NodeConfig) Node {
	info := NodeInfo{
		ID: config.ID,
	}

	node := &DockerNode{
		config:  config,
		adapter: &a,
		info:    info,
	}
	return node
}

// Snapshot returns a snapshot of the adapter
func (a DockerAdapter) Snapshot() AdapterSnapshot {
	return AdapterSnapshot{
		Type:   "docker",
		Config: a.config,
	}
}

// Info returns the node status
func (n *DockerNode) Info() NodeInfo {
	return n.info
}

// Start starts the node
func (n *DockerNode) Start() error {
	var err error
	defer func() {
		if err != nil {
			log.Error("Stopping node due to errors", "err", err)
			if err := n.Stop(); err != nil {
				log.Error("Failed stopping node", "err", err)
			}
		}
	}()

	// Define arguments
	args := []string{}

	// Append user defined arguments
	args = append(args, n.config.Args...)

	// Append network ports arguments
	args = append(args, "--pprofport", strconv.Itoa(dockerPProfPort))
	args = append(args, "--bzzport", strconv.Itoa(dockerHTTPPort))
	args = append(args, "--ws")
	// TODO: Can we get the APIs from somewhere instead of hardcoding them here?
	args = append(args, "--wsapi", "admin,net,debug,bzz,accounting,hive")
	args = append(args, "--wsport", strconv.Itoa(dockerWebsocketPort))
	args = append(args, "--wsaddr", "0.0.0.0")
	args = append(args, "--wsorigins", "*")
	args = append(args, "--port", strconv.Itoa(dockerP2PPort))
	args = append(args, "--natif", "eth0")

	// Start the node via a container
	ctx := context.Background()
	dockercli := n.adapter.client

	resp, err := dockercli.ContainerCreate(ctx, &container.Config{
		Image: n.adapter.image,
		Cmd:   args,
		Env:   n.config.Env,
	}, &container.HostConfig{}, nil, n.containerName())
	if err != nil {
		return fmt.Errorf("failed to create container %s: %v", n.containerName(), err)
	}

	if err := dockercli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %v", n.containerName(), err)
	}

	// Get container logs
	if n.config.Stderr != nil {
		go func() {
			// Stderr
			stderr, err := dockercli.ContainerLogs(context.Background(), n.containerName(), types.ContainerLogsOptions{
				ShowStderr: true,
				ShowStdout: false,
				Follow:     true,
			})
			if err != nil && err != io.EOF {
				log.Error("Error getting stderr container logs", "err", err)
			}
			defer stderr.Close()
			if _, err := io.Copy(n.config.Stderr, stderr); err != nil && err != io.EOF {
				log.Error("Error writing stderr container logs", "err", err)
			}
		}()
	}
	if n.config.Stdout != nil {
		go func() {
			// Stdout
			stdout, err := dockercli.ContainerLogs(context.Background(), n.containerName(), types.ContainerLogsOptions{
				ShowStderr: false,
				ShowStdout: true,
				Follow:     true,
			})
			if err != nil && err != io.EOF {
				log.Error("Error getting stdout container logs", "err", err)
			}
			defer stdout.Close()
			if _, err := io.Copy(n.config.Stdout, stdout); err != nil && err != io.EOF {
				log.Error("Error writing stdout container logs", "err", err)
			}
		}()
	}

	// Get container info
	cinfo := types.ContainerJSON{}

	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		cinfo, err = dockercli.ContainerInspect(ctx, n.containerName())
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("could not get container info: %v", err)
	}

	// Get the container IP addr
	n.ipAddr = cinfo.NetworkSettings.IPAddress

	// Wait for the node to start
	client, err := n.rpcClient()
	if err != nil {
		return err
	}
	defer client.Close()

	var swarminfo swarm.Info
	err = client.Call(&swarminfo, "bzz_info")
	if err != nil {
		return fmt.Errorf("could not get info via rpc call. node %s: %v", n.config.ID, err)
	}

	var p2pinfo p2p.NodeInfo
	err = client.Call(&p2pinfo, "admin_nodeInfo")
	if err != nil {
		return fmt.Errorf("could not get info via rpc call. node %s: %v", n.config.ID, err)
	}

	n.info = NodeInfo{
		ID:          n.config.ID,
		Enode:       strings.Replace(p2pinfo.Enode, "127.0.0.1", n.ipAddr, 1),
		BzzAddr:     swarminfo.BzzKey,
		RPCListen:   fmt.Sprintf("ws://%s:%d", n.ipAddr, dockerWebsocketPort),
		HTTPListen:  fmt.Sprintf("http://%s:%d", n.ipAddr, dockerHTTPPort),
		PprofListen: fmt.Sprintf("http://%s:%d", n.ipAddr, dockerPProfPort),
	}

	return nil
}

// Stop stops the node
func (n *DockerNode) Stop() error {
	cli := n.adapter.client

	var stopTimeout = 30 * time.Second
	err := cli.ContainerStop(context.Background(), n.containerName(), &stopTimeout)
	if err != nil {
		return fmt.Errorf("failed to stop container %s : %v", n.containerName(), err)
	}

	err = cli.ContainerRemove(context.Background(), n.containerName(), types.ContainerRemoveOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove container %s : %v", n.containerName(), err)
	}
	return nil
}

// Snapshot returns a snapshot of the node
func (n *DockerNode) Snapshot() (NodeSnapshot, error) {
	snap := NodeSnapshot{
		Config: n.config,
	}
	adapterSnap := n.adapter.Snapshot()
	snap.Adapter = &adapterSnap
	return snap, nil
}

func (n *DockerNode) containerName() string {
	return fmt.Sprintf("sim-docker-%s", n.config.ID)
}

func (n *DockerNode) rpcClient() (*rpc.Client, error) {
	var client *rpc.Client
	var err error
	wsAddr := fmt.Sprintf("ws://%s:%d", n.ipAddr, dockerWebsocketPort)
	for start := time.Now(); time.Since(start) < 30*time.Second; time.Sleep(50 * time.Millisecond) {
		client, err = rpc.Dial(wsAddr)
		if err == nil {
			break
		}
	}
	if client == nil {
		return nil, fmt.Errorf("could not establish rpc connection. node %s: %v", n.config.ID, err)
	}
	return client, nil
}

// buildImage builds a docker image and returns the image identifier (tag).
func buildImage(buildContext DockerBuildContext, deamonAddr string) (string, error) {
	// Connect to docker daemon
	c, err := client.NewClientWithOpts(
		client.WithHost(deamonAddr),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return "", fmt.Errorf("could not create docker client: %v", err)
	}
	defer c.Close()
	// Use directory for build context
	ctx, err := archive.TarWithOptions(buildContext.Directory, &archive.TarOptions{})
	if err != nil {
		return "", err
	}

	// Default image tag
	imageTag := "sim-docker:latest"

	// Use a tag if one is defined
	if buildContext.Tag != "" {
		imageTag = buildContext.Tag
	}

	// Build image
	opts := types.ImageBuildOptions{
		SuppressOutput: false,
		PullParent:     true,
		Tags:           []string{imageTag},
		Dockerfile:     buildContext.Dockerfile,
	}

	buildResp, err := c.ImageBuild(context.Background(), ctx, opts)
	if err != nil {
		return "", fmt.Errorf("build error: %v", err)
	}

	// Parse build output
	d := json.NewDecoder(buildResp.Body)
	var event *jsonmessage.JSONMessage
	for {
		if err := d.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		log.Info("Docker build", "msg", event.Stream)
		if event.Error != nil {
			log.Error("Docker build error", "err", event.Error.Message)
			return "", fmt.Errorf("failed to build docker image: %v", event.Error)
		}
	}
	return imageTag, nil
}
