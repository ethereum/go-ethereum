package simulation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethersphere/swarm"
	"github.com/ethersphere/swarm/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesAdapter can manage nodes on a kubernetes cluster
type KubernetesAdapter struct {
	client *kubernetes.Clientset
	config KubernetesAdapterConfig
	image  string
	proxy  string
}

// KubernetesAdapterConfig is the configuration provided to a KubernetesAdapter
type KubernetesAdapterConfig struct {
	// KubeConfigPath is the path to your kubernetes configuration path
	KubeConfigPath string `json:"kubeConfigPath"`
	// Namespace is the kubernetes namespaces where the pods should be running
	Namespace string `json:"namespace"`
	// BuildContext can be used to build a docker image
	// from a Dockerfile and a context directory
	BuildContext *KubernetesBuildContext `json:"build,omitempty"`
	// DockerImage points to an existing docker image
	// e.g. ethersphere/swarm:latest
	DockerImage string `json:"image,omitempty"`
}

// KubernetesBuildContext defines the build context to build
// local docker images
type KubernetesBuildContext struct {
	// Dockefile is the path to the dockerfile
	Dockerfile string `json:"dockerfile"`
	// Directory is the directory that will be used
	// in the context of a docker build
	Directory string `json:"dir"`
	// Tag is used to tag the image
	Tag string `json:"tag"`
	// Registry is the image registry where the image will be pushed to
	Registry string `json:"registry"`
	// Username is the user used to push the image to the registry
	Username string `json:"username"`
	// Password is the password of the user that is used to push the image
	// to the registry
	Password string `json:"-"`
}

// ImageTag is the full image tag, including the registry
func (bc *KubernetesBuildContext) ImageTag() string {
	return fmt.Sprintf("%s/%s", bc.Registry, bc.Tag)
}

// DefaultKubernetesAdapterConfig uses the default ~/.kube/config
// to discover the kubernetes clusters. It also uses the "default" namespace.
func DefaultKubernetesAdapterConfig() KubernetesAdapterConfig {
	kubeconfig := filepath.Join(homeDir(), ".kube", "config")
	return KubernetesAdapterConfig{
		KubeConfigPath: kubeconfig,
		Namespace:      "default",
	}
}

// IsKubernetesAvailable checks if a kubernetes configuration file exists
func IsKubernetesAvailable(kubeConfigPath string) bool {
	k8scfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return false
	}
	_, err = kubernetes.NewForConfig(k8scfg)
	return err == nil
}

// NewKubernetesAdapter creates a KubernetesAdpater by receiving a KubernetesAdapterConfig
func NewKubernetesAdapter(config KubernetesAdapterConfig) (*KubernetesAdapter, error) {
	if config.DockerImage != "" && config.BuildContext != nil {
		return nil, fmt.Errorf("only one can be defined: BuildContext (%v) or DockerImage(%s)",
			config.BuildContext, config.DockerImage)
	}

	if config.DockerImage == "" && config.BuildContext == nil {
		return nil, errors.New("required: Dockerfile or DockerImage")
	}

	// Define k8s client configuration
	k8scfg, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("could not start k8s client from config: %v", err)

	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(k8scfg)
	if err != nil {
		return nil, fmt.Errorf("could not create clientset: %v", err)
	}

	// Figure out which docker image should be used
	image := config.DockerImage

	// Build and push container image
	if config.BuildContext != nil {
		var err error
		// Build image
		image, err = buildImage(DockerBuildContext{
			Dockerfile: config.BuildContext.Dockerfile,
			Directory:  config.BuildContext.Directory,
			Tag:        config.BuildContext.ImageTag(),
		}, DefaultDockerAdapterConfig().DaemonAddr)
		if err != nil {
			return nil, fmt.Errorf("could not build the docker image: %v", err)
		}

		// Push image
		dockerClient, err := client.NewClientWithOpts(
			client.WithHost(client.DefaultDockerHost),
			client.WithAPIVersionNegotiation(),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create the docker client: %v", err)
		}

		authConfig := types.AuthConfig{
			Username: config.BuildContext.Username,
			Password: config.BuildContext.Password,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return nil, errors.New("failed marshaling the authentication parameters")
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)

		out, err := dockerClient.ImagePush(
			context.Background(),
			config.BuildContext.ImageTag(),
			types.ImagePushOptions{
				RegistryAuth: authStr,
			})
		if err != nil {
			return nil, fmt.Errorf("failed to push image: %v", err)
		}
		defer out.Close()
		if _, err := io.Copy(os.Stdout, out); err != nil && err != io.EOF {
			log.Error("Error pushing docker image", "err", err)
		}
	}

	// Setup proxy to access pods
	server, err := newProxyServer(k8scfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy: %v", err)
	}

	l, err := server.Listen("127.0.0.1", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to start proxy: %v", err)
	}
	go func() {
		if err := server.ServeOnListener(l); err != nil {
			log.Error("Kubernetes dapater proxy failed:", "err", err.Error())
		}
	}()

	// Return adapter
	return &KubernetesAdapter{
		client: clientset,
		image:  image,
		config: config,
		proxy:  l.Addr().String(),
	}, nil
}

// NewNode creates a new node
func (a KubernetesAdapter) NewNode(config NodeConfig) Node {
	info := NodeInfo{
		ID: config.ID,
	}
	node := &KubernetesNode{
		config:  config,
		adapter: &a,
		info:    info,
	}
	return node
}

// Snapshot returns a snapshot of the Adapter
func (a KubernetesAdapter) Snapshot() AdapterSnapshot {
	return AdapterSnapshot{
		Type:   "kubernetes",
		Config: a.config,
	}
}

// KubernetesNode is a node that was started via the KubernetesAdapter
type KubernetesNode struct {
	config  NodeConfig
	adapter *KubernetesAdapter
	info    NodeInfo
}

// Info returns the node info
func (n *KubernetesNode) Info() NodeInfo {
	return n.info
}

// Start starts the node
func (n *KubernetesNode) Start() error {
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
	args = append(args, "--nat", "ip:$(POD_IP)")

	// Build environment variables
	env := []v1.EnvVar{
		{
			// POD_IP is useful for setting the NAT config: e.g. `--nat ip:$POD_IP`
			Name: "POD_IP",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
	}
	for _, e := range n.config.Env {
		var name, value string
		s := strings.SplitN(e, "=", 1)
		name = s[0]
		if len(s) > 1 {
			value = s[1]
		}
		env = append(env, v1.EnvVar{
			Name:  name,
			Value: value,
		})
	}

	adapter := n.adapter

	// Create Kubernetes Pod
	podRequest := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: n.podName(),
			Labels: map[string]string{
				"app": "simulation",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  n.podName(),
					Image: adapter.image,
					Args:  args,
					Env:   env,
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceMemory: resource.MustParse("400Mi"),
						},
					},
				},
			},
		},
	}
	pod, err := adapter.client.CoreV1().Pods(adapter.config.Namespace).Create(podRequest)
	if err != nil {
		return fmt.Errorf("failed to create pod: %v", err)
	}

	// Wait for pod
	start := time.Now()
	for {
		log.Debug("Waiting for pod", "pod", n.podName())
		pod, err := adapter.client.CoreV1().Pods(adapter.config.Namespace).Get(n.podName(), metav1.GetOptions{})
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if pod.Status.Phase == v1.PodRunning {
			break
		}
		if time.Since(start) > 5*time.Minute {
			return errors.New("timeout waiting for pod")
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Get logs
	logOpts := &v1.PodLogOptions{
		Container: n.podName(),
		Follow:    true,
		Previous:  false,
	}
	req := adapter.client.CoreV1().Pods(adapter.config.Namespace).GetLogs(n.podName(), logOpts)
	readCloser, err := req.Stream()
	if err != nil {
		return fmt.Errorf("could not get logs: %v", err)
	}

	go func() {
		defer readCloser.Close()
		if _, err := io.Copy(n.config.Stderr, readCloser); err != nil && err != io.EOF {
			log.Error("Error writing pod logs", "pod", pod.Name, "err", err)
		}
	}()

	// Wait for the node to start
	var client *rpc.Client
	wsAddr := fmt.Sprintf("ws://%s/api/v1/namespaces/%s/pods/%s:%d/proxy",
		adapter.proxy, adapter.config.Namespace, n.podName(), dockerWebsocketPort)

	for start := time.Now(); time.Since(start) < 30*time.Second; time.Sleep(50 * time.Millisecond) {
		client, err = rpc.Dial(wsAddr)
		if err == nil {
			break
		}
	}
	if client == nil {
		return fmt.Errorf("could not establish rpc connection. node %s: %v", n.config.ID, err)
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
		ID:        n.config.ID,
		Enode:     p2pinfo.Enode,
		BzzAddr:   swarminfo.BzzKey,
		RPCListen: wsAddr,
		HTTPListen: fmt.Sprintf("http://%s/api/v1/namespaces/%s/pods/%s:%d/proxy",
			adapter.proxy, adapter.config.Namespace, n.podName(), dockerHTTPPort),
		PprofListen: fmt.Sprintf("http://%s/api/v1/namespaces/%s/pods/%s:%d/proxy",
			adapter.proxy, adapter.config.Namespace, n.podName(), dockerPProfPort),
	}

	return nil
}

// Stop stops the node
func (n *KubernetesNode) Stop() error {
	adapter := n.adapter

	gracePeriod := int64(30)

	deleteOpts := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}
	err := adapter.client.CoreV1().Pods(adapter.config.Namespace).Delete(n.podName(), deleteOpts)
	if err != nil {
		return fmt.Errorf("could not delete pod: %v", err)
	}
	return nil
}

// Snapshot returns a snapshot of the node
func (n *KubernetesNode) Snapshot() (NodeSnapshot, error) {
	snap := NodeSnapshot{
		Config: n.config,
	}
	adapterSnap := n.adapter.Snapshot()
	snap.Adapter = &adapterSnap
	return snap, nil
}

func (n *KubernetesNode) podName() string {
	return fmt.Sprintf("sim-k8s-%s", n.config.ID)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// proxyServer is a http.Handler which proxies Kubernetes APIs to remote API server.
type proxyServer struct {
	handler http.Handler
}

// Listen is a simple wrapper around net.Listen.
func (s *proxyServer) Listen(address string, port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
}

// ServeOnListener starts the server using given listener, loops forever.
func (s *proxyServer) ServeOnListener(l net.Listener) error {
	server := http.Server{
		Handler: s.handler,
	}
	return server.Serve(l)
}

func (s *proxyServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	s.handler.ServeHTTP(rw, req)
}

// newProxyServer creates a proxy server that can be used to proxy to the kubernetes API
func newProxyServer(cfg *rest.Config) (*proxyServer, error) {
	target, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	transport, err := rest.TransportFor(cfg)
	if err != nil {
		return nil, err
	}

	proxy.Transport = transport

	return &proxyServer{
		handler: proxy,
	}, nil
}
