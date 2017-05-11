package simulations

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/websocket"
)

// DefaultClient is the default simulation API client which expects the API
// to be running at http://localhost:8888
var DefaultClient = NewClient("http://localhost:8888")

// Client is a client for the simulation HTTP API which supports creating
// and managing simulation networks
type Client struct {
	URL string

	client *http.Client
}

// NewClient returns a new simulation API client
func NewClient(url string) *Client {
	return &Client{
		URL:    url,
		client: http.DefaultClient,
	}
}

// GetNetworks returns a list of simulations networks
func (c *Client) GetNetworks() ([]*Network, error) {
	var networks []*Network
	return networks, c.Get("/networks", &networks)
}

// CreateNetwork creates a new simulation network
func (c *Client) CreateNetwork(config *NetworkConfig) (*Network, error) {
	network := &Network{}
	return network, c.Post("/networks", config, network)
}

// GetNetwork returns details of a network
func (c *Client) GetNetwork(networkID string) (*Network, error) {
	network := &Network{}
	return network, c.Get(fmt.Sprintf("/networks/%s", networkID), network)
}

// SubscribeNetwork subscribes to network events which are sent from the server
// as a server-sent-events stream
func (c *Client) SubscribeNetwork(networkID string, events chan *Event) (event.Subscription, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/networks/%s/events", c.URL, networkID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		response, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		return nil, fmt.Errorf("unexpected HTTP status: %s: %s", res.Status, response)
	}

	// define a producer function to pass to event.Subscription
	// which reads server-sent events from res.Body and sends
	// them to the events channel
	producer := func(stop <-chan struct{}) error {
		defer res.Body.Close()

		// read lines from res.Body in a goroutine so that we are
		// always reading from the stop channel
		lines := make(chan string)
		errC := make(chan error, 1)
		go func() {
			s := bufio.NewScanner(res.Body)
			for s.Scan() {
				select {
				case lines <- s.Text():
				case <-stop:
					return
				}
			}
			errC <- s.Err()
		}()

		// detect any lines which start with "data:", decode the data
		// into an event and send it to the events channel
		for {
			select {
			case line := <-lines:
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				event := &Event{}
				if err := json.Unmarshal([]byte(data), event); err != nil {
					return fmt.Errorf("error decoding SSE event: %s", err)
				}
				select {
				case events <- event:
				case <-stop:
					return nil
				}
			case err := <-errC:
				return err
			case <-stop:
				return nil
			}
		}
	}

	return event.NewSubscription(producer), nil
}

// GetNodes returns all nodes which exist in a network
func (c *Client) GetNodes(networkID string) ([]*p2p.NodeInfo, error) {
	var nodes []*p2p.NodeInfo
	return nodes, c.Get(fmt.Sprintf("/networks/%s/nodes", networkID), &nodes)
}

// CreateNode creates a node in a network using the given configuration
func (c *Client) CreateNode(networkID string, config *adapters.NodeConfig) (*p2p.NodeInfo, error) {
	node := &p2p.NodeInfo{}
	return node, c.Post(fmt.Sprintf("/networks/%s/nodes", networkID), config, node)
}

// GetNode returns details of a node
func (c *Client) GetNode(networkID, nodeID string) (*p2p.NodeInfo, error) {
	node := &p2p.NodeInfo{}
	return node, c.Get(fmt.Sprintf("/networks/%s/nodes/%s", networkID, nodeID), node)
}

// StartNode starts a node
func (c *Client) StartNode(networkID, nodeID string) error {
	return c.Post(fmt.Sprintf("/networks/%s/nodes/%s/start", networkID, nodeID), nil, nil)
}

// StopNode stops a node
func (c *Client) StopNode(networkID, nodeID string) error {
	return c.Post(fmt.Sprintf("/networks/%s/nodes/%s/stop", networkID, nodeID), nil, nil)
}

// ConnectNode connects a node to a peer node
func (c *Client) ConnectNode(networkID, nodeID, peerID string) error {
	return c.Post(fmt.Sprintf("/networks/%s/nodes/%s/conn/%s", networkID, nodeID, peerID), nil, nil)
}

// DisconnectNode disconnects a node from a peer node
func (c *Client) DisconnectNode(networkID, nodeID, peerID string) error {
	return c.Delete(fmt.Sprintf("/networks/%s/nodes/%s/conn/%s", networkID, nodeID, peerID))
}

// RPCClient returns an RPC client connected to a node
func (c *Client) RPCClient(ctx context.Context, networkID, nodeID string) (*rpc.Client, error) {
	baseURL := strings.Replace(c.URL, "http", "ws", 1)
	return rpc.DialWebsocket(ctx, fmt.Sprintf("%s/networks/%s/nodes/%s/rpc", baseURL, networkID, nodeID), "")
}

// Get performs a HTTP GET request decoding the resulting JSON response
// into "out"
func (c *Client) Get(path string, out interface{}) error {
	return c.Send("GET", path, nil, out)
}

// Post performs a HTTP POST request sending "in" as the JSON body and
// decoding the resulting JSON response into "out"
func (c *Client) Post(path string, in, out interface{}) error {
	return c.Send("POST", path, in, out)
}

// Delete performs a HTTP DELETE request
func (c *Client) Delete(path string) error {
	return c.Send("DELETE", path, nil, nil)
}

// Send performs a HTTP request, sending "in" as the JSON request body and
// decoding the JSON response into "out"
func (c *Client) Send(method, path string, in, out interface{}) error {
	var body []byte
	if in != nil {
		var err error
		body, err = json.Marshal(in)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, c.URL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		response, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("unexpected HTTP status: %s: %s", res.Status, response)
	}
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return err
		}
	}
	return nil
}

// ServerConfig is the configuration used to start an API server
type ServerConfig struct {
	// Adapter is the NodeAdapter to use when creating new networks
	Adapter adapters.NodeAdapter

	// Mocker is the function which will be called when a client sends a
	// POST request to /networks/<netid>/mock and is expected to
	// generate some mock events in the network
	Mocker func(*Network)
}

// Server is an HTTP server providing an API to create and manage simulation
// networks
type Server struct {
	ServerConfig

	router   *httprouter.Router
	networks map[string]*Network
	mtx      sync.Mutex
}

// NewServer returns a new simulation API server
func NewServer(config *ServerConfig) *Server {
	if config.Adapter == nil {
		panic("Adapter not set")
	}

	s := &Server{
		ServerConfig: *config,
		router:       httprouter.New(),
		networks:     make(map[string]*Network),
	}

	s.POST("/networks", s.CreateNetwork)
	s.GET("/networks", s.GetNetworks)
	s.GET("/networks/:netid", s.GetNetwork)
	s.GET("/networks/:netid/events", s.StreamNetworkEvents)
	s.POST("/networks/:netid/mock", s.StartMocker)
	s.POST("/networks/:netid/nodes", s.CreateNode)
	s.GET("/networks/:netid/nodes", s.GetNodes)
	s.GET("/networks/:netid/nodes/:nodeid", s.GetNode)
	s.POST("/networks/:netid/nodes/:nodeid/start", s.StartNode)
	s.POST("/networks/:netid/nodes/:nodeid/stop", s.StopNode)
	s.POST("/networks/:netid/nodes/:nodeid/conn/:peerid", s.ConnectNode)
	s.DELETE("/networks/:netid/nodes/:nodeid/conn/:peerid", s.DisconnectNode)
	s.GET("/networks/:netid/nodes/:nodeid/rpc", s.NodeRPC)

	return s
}

// CreateNetwork creates a new simulation network
func (s *Server) CreateNetwork(w http.ResponseWriter, req *http.Request) {
	config := &NetworkConfig{}
	if err := json.NewDecoder(req.Body).Decode(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	network, err := func() (*Network, error) {
		s.mtx.Lock()
		defer s.mtx.Unlock()
		if config.Id == "" {
			config.Id = fmt.Sprintf("net%d", len(s.networks)+1)
		}
		if _, exists := s.networks[config.Id]; exists {
			return nil, fmt.Errorf("network exists: %s", config.Id)
		}
		network := NewNetwork(s.Adapter, config)
		s.networks[config.Id] = network
		return network, nil
	}()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.JSON(w, http.StatusCreated, network)
}

// GetNetworks returns a list of simulations networks
func (s *Server) GetNetworks(w http.ResponseWriter, req *http.Request) {
	s.mtx.Lock()
	networks := make([]*Network, 0, len(s.networks))
	for _, network := range s.networks {
		networks = append(networks, network)
	}
	s.mtx.Unlock()

	s.JSON(w, http.StatusOK, networks)
}

// GetNetwork returns details of a network
func (s *Server) GetNetwork(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)

	s.JSON(w, http.StatusOK, network)
}

func (s *Server) StartMocker(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)

	if s.Mocker == nil {
		http.Error(w, "mocker not configured", http.StatusInternalServerError)
		return
	}

	go s.Mocker(network)

	w.WriteHeader(http.StatusOK)
}

// StreamNetworkEvents streams network events as a server-sent-events stream
func (s *Server) StreamNetworkEvents(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)

	events := make(chan *Event)
	sub := network.events.Subscribe(events)
	defer sub.Unsubscribe()

	// stop the stream if the client goes away
	var clientGone <-chan bool
	if cn, ok := w.(http.CloseNotifier); ok {
		clientGone = cn.CloseNotify()
	}

	// write writes the given event and data to the stream like:
	//
	// event: <event>
	// data: <data>
	//
	write := func(event, data string) {
		fmt.Fprintf(w, "event: %s\n", event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		if fw, ok := w.(http.Flusher); ok {
			fw.Flush()
		}
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if fw, ok := w.(http.Flusher); ok {
		fw.Flush()
	}
	for {
		select {
		case event := <-events:
			data, err := json.Marshal(event)
			if err != nil {
				write("error", err.Error())
				return
			}
			write("network", string(data))
		case <-clientGone:
			return
		}
	}
}

// CreateNode creates a node in a network using the given configuration
func (s *Server) CreateNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)

	config := adapters.RandomNodeConfig()
	err := json.NewDecoder(req.Body).Decode(config)
	if err != nil && err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	node, err := network.NewNodeWithConfig(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusCreated, node.NodeInfo())
}

// GetNodes returns all nodes which exist in a network
func (s *Server) GetNodes(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)

	nodes := network.GetNodes()

	infos := make([]*p2p.NodeInfo, len(nodes))
	for i, node := range nodes {
		infos[i] = node.NodeInfo()
	}

	s.JSON(w, http.StatusOK, infos)
}

// GetNode returns details of a node
func (s *Server) GetNode(w http.ResponseWriter, req *http.Request) {
	node := req.Context().Value("node").(*Node)

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

// StartNode starts a node
func (s *Server) StartNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)
	node := req.Context().Value("node").(*Node)

	if err := network.Start(node.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

// StopNode stops a node
func (s *Server) StopNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)
	node := req.Context().Value("node").(*Node)

	if err := network.Stop(node.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

// ConnectNode connects a node to a peer node
func (s *Server) ConnectNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)
	node := req.Context().Value("node").(*Node)
	peer := req.Context().Value("peer").(*Node)

	if err := network.Connect(node.ID(), peer.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

// DisconnectNode disconnects a node from a peer node
func (s *Server) DisconnectNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)
	node := req.Context().Value("node").(*Node)
	peer := req.Context().Value("peer").(*Node)

	if err := network.Disconnect(node.ID(), peer.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

// NodeRPC proxies node RPC requests via a WebSocket connection
func (s *Server) NodeRPC(w http.ResponseWriter, req *http.Request) {
	node := req.Context().Value("node").(*Node)

	handler := func(conn *websocket.Conn) {
		node.ServeRPC(conn)
	}

	websocket.Server{Handler: handler}.ServeHTTP(w, req)
}

// ServeHTTP implements the http.Handler interface by delegating to the
// underlying httprouter.Router
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.router.ServeHTTP(w, req)
}

// GET registers a handler for GET requests to a particular path
func (s *Server) GET(path string, handle http.HandlerFunc) {
	s.router.GET(path, s.wrapHandler(handle))
}

// POST registers a handler for POST requests to a particular path
func (s *Server) POST(path string, handle http.HandlerFunc) {
	s.router.POST(path, s.wrapHandler(handle))
}

// DELETE registers a handler for DELETE requests to a particular path
func (s *Server) DELETE(path string, handle http.HandlerFunc) {
	s.router.DELETE(path, s.wrapHandler(handle))
}

// JSON sends "data" as a JSON HTTP response
func (s *Server) JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// wrapHandler returns a httprouter.Handle which wraps a http.HandlerFunc by
// populating request.Context with any objects from the URL params
func (s *Server) wrapHandler(handler http.HandlerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		ctx := context.Background()

		var network *Network
		if id := params.ByName("netid"); id != "" {
			s.mtx.Lock()
			var ok bool
			network, ok = s.networks[id]
			s.mtx.Unlock()
			if !ok {
				http.NotFound(w, req)
				return
			}
			ctx = context.WithValue(ctx, "network", network)
		}

		if id := params.ByName("nodeid"); id != "" {
			if network == nil {
				http.NotFound(w, req)
				return
			}
			var node *Node
			if nodeID, err := discover.HexID(id); err == nil {
				node = network.GetNode(&adapters.NodeId{NodeID: nodeID})
			} else {
				node = network.GetNodeByName(id)
			}
			if node == nil {
				http.NotFound(w, req)
				return
			}
			ctx = context.WithValue(ctx, "node", node)
		}

		if id := params.ByName("peerid"); id != "" {
			if network == nil {
				http.NotFound(w, req)
				return
			}
			var peer *Node
			if peerID, err := discover.HexID(id); err == nil {
				peer = network.GetNode(&adapters.NodeId{NodeID: peerID})
			} else {
				peer = network.GetNodeByName(id)
			}
			if peer == nil {
				http.NotFound(w, req)
				return
			}
			ctx = context.WithValue(ctx, "peer", peer)
		}

		handler(w, req.WithContext(ctx))
	}
}
