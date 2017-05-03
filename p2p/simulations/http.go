package simulations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/julienschmidt/httprouter"
	"github.com/pborman/uuid"
)

type ServerConfig struct {
	Adapter adapters.NodeAdapter
	Mocker  func(*Network)
}

type Server struct {
	ServerConfig

	router   *httprouter.Router
	networks map[string]*Network
	mtx      sync.Mutex
}

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
	s.POST("/networks/:netid/mock", s.StartMocker)
	s.POST("/networks/:netid/nodes", s.CreateNode)
	s.GET("/networks/:netid/nodes", s.GetNodes)
	s.GET("/networks/:netid/nodes/:nodeid", s.GetNode)
	s.POST("/networks/:netid/nodes/:nodeid/start", s.StartNode)
	s.POST("/networks/:netid/nodes/:nodeid/stop", s.StopNode)
	s.POST("/networks/:netid/nodes/:nodeid/conn/:peerid", s.ConnectNode)
	s.DELETE("/networks/:netid/nodes/:nodeid/conn/:peerid", s.DisconnectNode)

	return s
}

func (s *Server) CreateNetwork(w http.ResponseWriter, req *http.Request) {
	config := &NetworkConfig{}
	if err := json.NewDecoder(req.Body).Decode(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if config.Id == "" {
		config.Id = uuid.NewRandom().String()
	}

	err := func() error {
		s.mtx.Lock()
		defer s.mtx.Unlock()
		if _, exists := s.networks[config.Id]; exists {
			return fmt.Errorf("network exists: %s", config.Id)
		}
		network := NewNetwork(s.Adapter, config)
		s.networks[config.Id] = network
		return nil
	}()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.JSON(w, http.StatusCreated, config)
}

func (s *Server) GetNetworks(w http.ResponseWriter, req *http.Request) {
	s.mtx.Lock()
	networks := make([]NetworkConfig, 0, len(s.networks))
	for _, network := range s.networks {
		config := network.Config()
		networks = append(networks, *config)
	}
	s.mtx.Unlock()

	s.JSON(w, http.StatusOK, networks)
}

func (s *Server) GetNetwork(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)

	if req.Header.Get("Accept") == "text/event-stream" {
		s.streamNetworkEvents(network, w)
		return
	}

	s.JSON(w, http.StatusOK, network.Config())
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

func (s *Server) streamNetworkEvents(network *Network, w http.ResponseWriter) {
	sub := network.events.Subscribe(ConnectivityAllEvents...)
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

	w.Header().Set("Content-Type", "text/event-stream")
	ch := sub.Chan()
	for {
		select {
		case event := <-ch:
			// convert the event to a SimUpdate
			update, err := NewSimUpdate(event)
			if err != nil {
				write("error", err.Error())
				return
			}
			data, err := json.Marshal(update)
			if err != nil {
				write("error", err.Error())
				return
			}
			write("simupdate", string(data))
		case <-clientGone:
			return
		}
	}
}

func (s *Server) CreateNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)

	config, err := network.NewNode()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusCreated, network.GetNode(config.Id))
}

func (s *Server) GetNodes(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)

	nodes := network.GetNodes()

	infos := make([]*p2p.NodeInfo, len(nodes))
	for i, node := range nodes {
		infos[i] = node.NodeInfo()
	}

	s.JSON(w, http.StatusOK, infos)
}

func (s *Server) GetNode(w http.ResponseWriter, req *http.Request) {
	node := req.Context().Value("node").(*Node)

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

func (s *Server) StartNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)
	node := req.Context().Value("node").(*Node)

	if err := network.Start(node.Id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

func (s *Server) StopNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)
	node := req.Context().Value("node").(*Node)

	if err := network.Stop(node.Id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

func (s *Server) ConnectNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)
	node := req.Context().Value("node").(*Node)
	peer := req.Context().Value("peer").(*Node)

	if err := network.Connect(node.Id, peer.Id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

func (s *Server) DisconnectNode(w http.ResponseWriter, req *http.Request) {
	network := req.Context().Value("network").(*Network)
	node := req.Context().Value("node").(*Node)
	peer := req.Context().Value("peer").(*Node)

	if err := network.Disconnect(node.Id, peer.Id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.JSON(w, http.StatusOK, node.NodeInfo())
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.router.ServeHTTP(w, req)
}

func (s *Server) GET(path string, handle http.HandlerFunc) {
	s.router.GET(path, s.wrapHandler(handle))
}

func (s *Server) POST(path string, handle http.HandlerFunc) {
	s.router.POST(path, s.wrapHandler(handle))
}

func (s *Server) DELETE(path string, handle http.HandlerFunc) {
	s.router.DELETE(path, s.wrapHandler(handle))
}

func (s *Server) JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

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
			node := network.GetNode(adapters.NewNodeIdFromHex(id))
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
			peer := network.GetNode(adapters.NewNodeIdFromHex(id))
			if peer == nil {
				http.NotFound(w, req)
				return
			}
			ctx = context.WithValue(ctx, "peer", peer)
		}

		handler(w, req.WithContext(ctx))
	}
}
