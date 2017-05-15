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

	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/node"
)

// DockerAdapter is a NodeAdapter which runs nodes inside Docker containers.
//
// A Docker image is built which contains the current binary at /bin/p2p-node
// which when executed runs the underlying service (see the description
// of the execP2PNode function for more details)
type DockerAdapter struct{}

// NewDockerAdapter builds the p2p-node Docker image containing the current
// binary and returns a DockerAdapter
func NewDockerAdapter() (*DockerAdapter, error) {
	if runtime.GOOS != "linux" {
		return nil, errors.New("DockerAdapter can only be used on Linux as it uses the current binary (which must be a Linux binary)")
	}

	if err := buildDockerImage(); err != nil {
		return nil, err
	}

	return &DockerAdapter{}, nil
}

// Name returns the name of the adapter for logging purpoeses
func (d *DockerAdapter) Name() string {
	return "docker-adapter"
}

// NewNode returns a new DockerNode using the given config
func (d *DockerAdapter) NewNode(config *NodeConfig) (Node, error) {
	for _, name := range config.Services {
		if _, exists := serviceFuncs[name]; !exists {
			return nil, fmt.Errorf("unknown node service %q", name)
		}
	}

	// generate the config
	conf := &execNodeConfig{
		Stack: node.DefaultConfig,
		Node:  config,
	}
	conf.Stack.DataDir = "/data"
	conf.Stack.P2P.EnableMsgEvents = true
	conf.Stack.P2P.NoDiscovery = true
	conf.Stack.P2P.NAT = nil

	node := &DockerNode{
		ExecNode: ExecNode{
			ID:     config.Id,
			Config: conf,
			Services: config.Services,
		},
	}
	node.newCmd = node.dockerCommand
	return node, nil
}

// DockerNode wraps an ExecNode but exec's the current binary in a docker
// container rather than locally
type DockerNode struct {
	ExecNode
}

// dockerCommand returns a command which exec's the binary in a docker
// container.
//
// It uses a shell so that we can pass the _P2P_NODE_CONFIG and _P2P_NODE_KEY
// environment variables to the container using the --env flag.
func (n *DockerNode) dockerCommand() *exec.Cmd {
	return exec.Command(
		"sh", "-c",
		fmt.Sprintf(
			`exec docker run --interactive --env _P2P_NODE_CONFIG="${_P2P_NODE_CONFIG}" --env _P2P_NODE_KEY="${_P2P_NODE_KEY}" %s p2p-node %s %s`,
			dockerImage, n.Services[0], n.ID.String(),
		),
	)
}

func (n *DockerNode) GetService(name string) node.Service {
	return nil
}

// dockerImage is the name of the docker image
const dockerImage = "p2p-node"

// buildDockerImage builds the docker image which is used to run devp2p nodes
// using docker.
//
// It adds the current binary as "p2p-node" so that it runs execP2PNode
// when executed.
func buildDockerImage() error {
	// create a directory to use as the docker build context
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
