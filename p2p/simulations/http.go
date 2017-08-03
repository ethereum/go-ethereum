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
	"strconv"
	"strings"

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

// GetNetwork returns details of the network
func (c *Client) GetNetwork() (*Network, error) {
	network := &Network{}
	return network, c.Get("/", network)
}

// StartNetwork starts all existing nodes in the simulation network
func (c *Client) StartNetwork() error {
	return c.Post("/start", nil, nil)
}

// StopNetwork stops all existing nodes in a simulation network
func (c *Client) StopNetwork() error {
	return c.Post("/stop", nil, nil)
}

// CreateSnapshot creates a network snapshot
func (c *Client) CreateSnapshot() (*Snapshot, error) {
	snap := &Snapshot{}
	return snap, c.Get("/snapshot", snap)
}

// LoadSnapshot loads a snapshot into the network
func (c *Client) LoadSnapshot(snap *Snapshot) error {
	return c.Post("/snapshot", snap, nil)
}

// SubscribeNetwork subscribes to network events which are sent from the server
// as a server-sent-events stream, optionally receiving events for existing
// nodes and connections

//this struct specifies the behavior of the SubscribeNetwork function:
type SubscribeOpts struct {
	//send the existing network (nodes,conns) over the event stream first
	Current bool
	//filter string defining which messages to filter (empty string for unfiltered stream)
	Filter string
}

func (c *Client) SubscribeNetwork(events chan *Event, opts SubscribeOpts) (event.Subscription, error) {
	var url string
	filter := opts.Filter
	current := opts.Current
	if filter != "" {
		url = fmt.Sprintf("%s/events?current=%t&filter=%s", c.URL, current, filter)
	} else {
		url = fmt.Sprintf("%s/events?current=%t", c.URL, current)
	}
	req, err := http.NewRequest("GET", url, nil)
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

// GetNodes returns all nodes which exist in the network
func (c *Client) GetNodes() ([]*p2p.NodeInfo, error) {
	var nodes []*p2p.NodeInfo
	return nodes, c.Get("/nodes", &nodes)
}

// CreateNode creates a node in the network using the given configuration
func (c *Client) CreateNode(config *adapters.NodeConfig) (*p2p.NodeInfo, error) {
	node := &p2p.NodeInfo{}
	return node, c.Post("/nodes", config, node)
}

// GetNode returns details of a node
func (c *Client) GetNode(nodeID string) (*p2p.NodeInfo, error) {
	node := &p2p.NodeInfo{}
	return node, c.Get(fmt.Sprintf("/nodes/%s", nodeID), node)
}

// StartNode starts a node
func (c *Client) StartNode(nodeID string) error {
	return c.Post(fmt.Sprintf("/nodes/%s/start", nodeID), nil, nil)
}

// StopNode stops a node
func (c *Client) StopNode(nodeID string) error {
	return c.Post(fmt.Sprintf("/nodes/%s/stop", nodeID), nil, nil)
}

// ConnectNode connects a node to a peer node
func (c *Client) ConnectNode(nodeID, peerID string) error {
	return c.Post(fmt.Sprintf("/nodes/%s/conn/%s", nodeID, peerID), nil, nil)
}

// DisconnectNode disconnects a node from a peer node
func (c *Client) DisconnectNode(nodeID, peerID string) error {
	return c.Delete(fmt.Sprintf("/nodes/%s/conn/%s", nodeID, peerID))
}

// RPCClient returns an RPC client connected to a node
func (c *Client) RPCClient(ctx context.Context, nodeID string) (*rpc.Client, error) {
	baseURL := strings.Replace(c.URL, "http", "ws", 1)
	return rpc.DialWebsocket(ctx, fmt.Sprintf("%s/nodes/%s/rpc", baseURL, nodeID), "")
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
	// Mocker is the function which will be called when a client sends a
	// POST request to /mock and is expected to generate some mock events
	// in the network
	Mocker func(*Network)
	// In case of multiple mockers, set the default here
	DefaultMockerID string
	// map of Mockers
	Mockers map[string]*MockerConfig
}

// Server is an HTTP server providing an API to manage a simulation network
type Server struct {
	ServerConfig

	router    *httprouter.Router
	network   *Network
}

// NewServer returns a new simulation API server
func NewServer(network *Network, config ServerConfig) *Server {
	s := &Server{
		ServerConfig: config,
		router:       httprouter.New(),
		network:      network,
	}

	s.OPTIONS("/", s.Options)
	s.GET("/", s.GetNetwork)
	s.POST("/start", s.StartNetwork)
	s.POST("/stop", s.StopNetwork)
	s.GET("/events", s.StreamNetworkEvents)
	s.GET("/snapshot", s.CreateSnapshot)
	s.POST("/snapshot", s.LoadSnapshot)
	s.POST("/mock/:mockid", s.StartMocker)
	s.GET("/mock", s.GetMocker)
	s.POST("/nodes", s.CreateNode)
	s.GET("/nodes", s.GetNodes)
	s.GET("/nodes/:nodeid", s.GetNode)
	s.POST("/nodes/:nodeid/start", s.StartNode)
	s.POST("/nodes/:nodeid/stop", s.StopNode)
	s.POST("/nodes/:nodeid/conn/:peerid", s.ConnectNode)
	s.DELETE("/nodes/:nodeid/conn/:peerid", s.DisconnectNode)
	s.GET("/nodes/:nodeid/rpc", s.NodeRPC)

	return s
}

// GetNetwork returns details of the network
func (s *Server) GetNetwork(w http.ResponseWriter, req *http.Request) {
	s.JSON(w, http.StatusOK, s.network)
}

// StartNetwork starts all nodes in the network
func (s *Server) StartNetwork(w http.ResponseWriter, req *http.Request) {
	if err := s.network.StartAll(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// StopNetwork stops all nodes in the network
func (s *Server) StopNetwork(w http.ResponseWriter, req *http.Request) {
	if err := s.network.StopAll(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

//Get the info for a particular mocker
func (s *Server) GetMocker(w http.ResponseWriter, req *http.Request) {
	m := make(map[string]string)

	for k, v := range s.Mockers {
		m[k] = v.Description
	}

	s.JSON(w, http.StatusOK, m)
}

func (s *Server) StartMocker(w http.ResponseWriter, req *http.Request) {
	mockerid := req.Context().Value("mock").(string)

	if len(s.Mockers) == 0 {
		//don't require a mocker to be present
		s.JSON(w, http.StatusNotModified, "No mocker configured")
		return
	}

	if mockerid == "default" {
		//choose the default mocker
		mockerid = s.DefaultMockerID
	}

	if mocker, ok := s.Mockers[mockerid]; ok {
		if mocker.Mocker == nil {
			http.Error(w, "mocker not configured", http.StatusInternalServerError)
			return
		}
		go mocker.Mocker(s.network)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "invalid mockerid provided", http.StatusBadRequest)
		return
	}
}

//This struct defines the filtering behavior
type MsgFilter struct {
	//the message protocol which will be filtered (allowed)
	Proto string
	//the message code which will be filtered
	Code uint64
}

// StreamNetworkEvents streams network events as a server-sent-events stream
func (s *Server) StreamNetworkEvents(w http.ResponseWriter, req *http.Request) {
	filters   := make(map[MsgFilter]struct{})
	wildcards := make(map[string]struct{})
	events := make(chan *Event)
	sub := s.network.events.Subscribe(events)
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
	writeEvent := func(event *Event) error {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		write("network", string(data))
		return nil
	}
	writeErr := func(err error) {
		write("error", err.Error())
	}

	// check if filtering has been requested
	if strfilter := req.URL.Query().Get("filter"); strfilter != "" {
		if err := s.setupFilter(strfilter, filters, wildcards); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "\n\n")
	if fw, ok := w.(http.Flusher); ok {
		fw.Flush()
	}

	// optionally send the existing nodes and connections
	if req.URL.Query().Get("current") == "true" {
		snap, err := s.network.Snapshot()
		if err != nil {
			writeErr(err)
			return
		}
		for _, node := range snap.Nodes {
			event := NewEvent(&node.Node)
			if err := writeEvent(event); err != nil {
				writeErr(err)
				return
			}
		}
		for _, conn := range snap.Conns {
			event := NewEvent(&conn)
			if err := writeEvent(event); err != nil {
				writeErr(err)
				return
			}
		}
	}

	for {
		select {
		case event := <-events:
			//filtering events only works for messages
			//if the message does not pass filter, break handling
			if event.Msg != nil && !s.msgMatchesFilter(event,filters,wildcards) {
				continue
			}
			if err := writeEvent(event); err != nil {
				writeErr(err)
				return
			}
		case <-clientGone:
			return
		}
	}
}

//This function sets up filtering messages, if a non-empty filter string is provided
//The filter string needs to be sent via the URL query string as 'filter=<strfilter>',
//where strfilter has the format: [PROTOCOL]:[[MESSAGE-CODE],]
//there needs to be at least one [PROTOCOL]:[MESSAGE-CODE] sequence; if multiple are present,
//they need to be separated by a '-'
//PROTOCOL must be a valid protocol for this to work (no validation here)
//A message-code of '*' or '-1' are considered wildcards
func (s *Server) setupFilter(strfilter string,filters map[MsgFilter]struct{}, wildcards map[string]struct{}) error {
	//split first all [PROTOCOL]:[MESSAGE-CODE] pairs into an array, separator being '-'
	arrfilter := strings.Split(strfilter, "-")
	//for each elem in array
	for _, f := range arrfilter {
		//get the parts: [0] is the protocol, [1] the codes, separated by ','
		fparts := strings.Split(f, ":")
		//thus, fparts must be exactly 2 in length and each elem can't be empty
		if len(fparts) != 2 || len(fparts[0]) == 0 || len(fparts[1]) == 0 {
			return fmt.Errorf("Invalid msg filter format provided", http.StatusBadRequest)
		}
		//get the protocol
		proto := fparts[0]
		//get the code(s), separated by ','
		strcodes := strings.Split(fparts[1], ",")
		for _, c := range strcodes {
			//code part '*' or '-1' is considered a wildcard, thus passing all codes of that protocol
			if fparts[1] == "*" || fparts[1] == "-1" {
				wildcards[fparts[0]] = struct{}{}
			} else {
				//no wildcards, thus parse code value
				code, err := strconv.ParseUint(c, 10, 64)
				if err != nil {
					return fmt.Errorf("Invalid msg code for filtering provided", http.StatusBadRequest)
				}
				//we got all filter info, set the filter on the server
				filters[MsgFilter{Proto: proto, Code: code}] = struct{}{}
			}
		}
	}
	return nil
}

//This function finally tries to match events to filters if filtering is enabled
//It compares the event Msg Protocol first to wildcards, then to codes
//Returns true if the event matches the filter (= event should be sent to client)
//or false if it doesn't match the filter (= event won't be sent to client)
//If no filter has been set, no messages will be delivered at all
func (s *Server) msgMatchesFilter(event *Event, filters map[MsgFilter]struct{}, wildcards map[string]struct{}) bool {
	//check if the Protocol matches a wildcard; in that case, don't even check for code;
	//allow all (for that protocol)
	_, ok := wildcards[event.Msg.Protocol]
	if !ok {
		//no wildcard matched; so check if this event matches protocol and specific code
		_, ok = filters[MsgFilter{event.Msg.Protocol, event.Msg.Code}]
		// filter matches, send the event
	}
	//true if event matched filter, false if not
	return ok
}

// CreateSnapshot creates a network snapshot
func (s *Server) CreateSnapshot(w http.ResponseWriter, req *http.Request) {
	snap, err := s.network.Snapshot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, snap)
}

// LoadSnapshot loads a snapshot into the network
func (s *Server) LoadSnapshot(w http.ResponseWriter, req *http.Request) {
	snap := &Snapshot{}
	if err := json.NewDecoder(req.Body).Decode(snap); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.network.Load(snap); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, s.network)
}

// CreateNode creates a node in the network using the given configuration
func (s *Server) CreateNode(w http.ResponseWriter, req *http.Request) {
	config := adapters.RandomNodeConfig()
	err := json.NewDecoder(req.Body).Decode(config)
	if err != nil && err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	node, err := s.network.NewNodeWithConfig(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusCreated, node.NodeInfo())
}

// GetNodes returns all nodes which exist in the network
func (s *Server) GetNodes(w http.ResponseWriter, req *http.Request) {
	nodes := s.network.GetNodes()

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
	node := req.Context().Value("node").(*Node)

	if err := s.network.Start(node.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

// StopNode stops a node
func (s *Server) StopNode(w http.ResponseWriter, req *http.Request) {
	node := req.Context().Value("node").(*Node)

	if err := s.network.Stop(node.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

// ConnectNode connects a node to a peer node
func (s *Server) ConnectNode(w http.ResponseWriter, req *http.Request) {
	node := req.Context().Value("node").(*Node)
	peer := req.Context().Value("peer").(*Node)

	if err := s.network.Connect(node.ID(), peer.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

// DisconnectNode disconnects a node from a peer node
func (s *Server) DisconnectNode(w http.ResponseWriter, req *http.Request) {
	node := req.Context().Value("node").(*Node)
	peer := req.Context().Value("peer").(*Node)

	if err := s.network.Disconnect(node.ID(), peer.ID()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

//Options
func (s *Server) Options(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	return
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

// OPTIONS registers a handler for OPTIONS requests to a particular path
func (s *Server) OPTIONS(path string, handle http.HandlerFunc) {
	s.router.OPTIONS("/*path", s.wrapHandler(handle))
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

		if id := params.ByName("nodeid"); id != "" {
			var node *Node
			if nodeID, err := discover.HexID(id); err == nil {
				node = s.network.GetNode(nodeID)
			} else {
				node = s.network.GetNodeByName(id)
			}
			if node == nil {
				http.NotFound(w, req)
				return
			}
			ctx = context.WithValue(ctx, "node", node)
		}

		if id := params.ByName("peerid"); id != "" {
			var peer *Node
			if peerID, err := discover.HexID(id); err == nil {
				peer = s.network.GetNode(peerID)
			} else {
				peer = s.network.GetNodeByName(id)
			}
			if peer == nil {
				http.NotFound(w, req)
				return
			}
			ctx = context.WithValue(ctx, "peer", peer)
		}

		if id := params.ByName("mockid"); id != "" {
			ctx = context.WithValue(ctx, "mock", id)
		}

		handler(w, req.WithContext(ctx))
	}
}
