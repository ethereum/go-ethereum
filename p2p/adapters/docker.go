package adapters

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/node"
)

// DockerNode is a NodeAdapter which wraps an ExecNode but exec's the current
// binary in a docker container rather than locally
type DockerNode struct {
	ExecNode
}

// NewDockerNode creates a new DockerNode, building the docker image if
// necessary
func NewDockerNode(id *NodeId, service string) (*DockerNode, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("NewDockerNode can only be used on Linux as it uses the current binary (which must be a Linux binary)")
	}

	if _, exists := serviceFuncs[service]; !exists {
		return nil, fmt.Errorf("unknown node service %q", service)
	}

	// build the docker image
	var err error
	dockerOnce.Do(func() {
		err = buildDockerImage()
	})
	if err != nil {
		return nil, err
	}

	// generate the config
	conf := node.DefaultConfig
	conf.DataDir = "/data"
	conf.P2P.NoDiscovery = true
	conf.P2P.NAT = nil

	node := &DockerNode{
		ExecNode: ExecNode{
			ID:      id,
			Service: service,
			Config:  &conf,
		},
	}
	node.newCmd = node.dockerCommand
	return node, nil
}

// dockerCommand returns a command which exec's the binary in a docker
// container.
//
// It uses a shell so that we can pass the _P2P_NODE_CONFIG environment
// variable to the container using the --env flag.
func (n *DockerNode) dockerCommand() *exec.Cmd {
	return exec.Command(
		"sh", "-c",
		fmt.Sprintf(
			`exec docker run --interactive --env _P2P_NODE_CONFIG="${_P2P_NODE_CONFIG}" %s p2p-node %s %s`,
			dockerImage, n.Service, n.ID.String(),
		),
	)
}

// dockerImage is the name of the docker image
const dockerImage = "p2p-node"

// dockerOnce is used to build the docker image only once
var dockerOnce sync.Once

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
