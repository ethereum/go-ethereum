package simulations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	
)

type returnHandler func(body io.Reader) (resp io.ReadSeeker, err error)

type ResourceHandler struct {
	Handle func(interface{}, *ResourceController) (interface{}, error)
	Type   reflect.Type
}

type ResourceHandlers struct {
	Create, Retrieve, Update, Destroy *ResourceHandler
}

type ResourceController struct {
	lock        sync.Mutex
	controllers map[string]Controller
	id          int
	methods     []string
	*ResourceHandlers
}

type NodeResult struct {
	Nodes []*Node	
}

type NodeIF struct {
 	One uint
 	Other uint
 	MessageType uint8
}
 
var methodsAvailable = []string{"POST", "GET", "PUT", "DELETE"}

func (self *ResourceHandlers) handler(method string) *ResourceHandler {
	var h *ResourceHandler
	switch method {
	case "POST":
		h = self.Create
	case "GET":
		h = self.Retrieve
	case "PUT":
		h = self.Update
	case "DELETE":
		h = self.Destroy
	case "OPTIONS":
		h = &ResourceHandler{
			Handle: func(msg interface{}, c *ResourceController) (interface{}, error) {
				return struct{}{}, nil
			},
		}
	}

	return h
}

func NewResourceContoller(c *ResourceHandlers) *ResourceController {
	var methods []string
	for _, method := range methodsAvailable {
		if c.handler(method) != nil {
			methods = append(methods, method)
		}
	}
	return &ResourceController{
		ResourceHandlers: c,
		controllers:      make(map[string]Controller),
		methods:          methods,
	}
}

var empty = struct{}{}

func NewSessionController() (*ResourceController, chan bool) {
	quitc := make(chan bool)
	return NewResourceContoller(
		&ResourceHandlers{

			Create: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					conf := msg.(*NetworkConfig)
					journal := NewJournal()
					net := NewNetwork(nil, &event.TypeMux{})
					net.SetNaf(net.NewGenericSimNode)
					m := NewNetworkController(conf, net.Events(), journal)
					if len(conf.Id) == 0 {
						conf.Id = fmt.Sprintf("%d", parent.id)
					}
					glog.V(6).Infof("new network controller on %v", conf.Id)
					if parent != nil {
						parent.SetResource(conf.Id, m)
					}
					parent.id++
					
					m.SetResource("debug", NewResourceContoller(
						&ResourceHandlers{
							Create: &ResourceHandler{
								Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
									journaldump := []string{}
									eventfmt := func(e *event.Event) bool {
										journaldump = append(journaldump, fmt.Sprintf("%v", e))
									return true
									}
									journal.Read(eventfmt)
									return struct{Results []string}{Results: journaldump,}, nil
								},
							},
						},
					))
					
					m.SetResource("node", NewResourceContoller(
						&ResourceHandlers{
							Create: &ResourceHandler{
								Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
									var nodeid *adapters.NodeId
									
									nodeid = adapters.RandomNodeId()
									
									net.NewNode(&NodeConfig{Id: nodeid})
									glog.V(6).Infof("added node %v to network %v", nodeid, net)
									
									return &NodeConfig{Id: nodeid}, nil
									
								},
							},
							Retrieve: &ResourceHandler{
								Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
									return &NodeResult{Nodes: net.Nodes}, nil
								},
							},
							Update: &ResourceHandler{
								Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
									var othernode *Node
									
									args := msg.(*NodeIF)
									onenode := net.Nodes[args.One - 1]
									
									if args.Other == 0 {
										if net.Start(onenode.Id) != nil {
											net.Stop(onenode.Id)	
										}
										return &NodeResult{Nodes: []*Node{onenode}}, nil
									} else {
										othernode = net.Nodes[args.Other - 1]
										net.Connect(onenode.Id, othernode.Id)
										return &NodeResult{Nodes: []*Node{onenode, othernode}}, nil
									}
								},
								Type: reflect.TypeOf(&NodeIF{}), // this is input not output param structure
							},
						},
					))
					
					return empty, nil
				},
				Type: reflect.TypeOf(&NetworkConfig{}),
			},

			Destroy: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					glog.V(6).Infof("destroy handler called")
					// this can quit the entire app (shut down the backend server)
					quitc <- true
					return empty, nil
				},
			},
		},
	), quitc
}

func (self *ResourceController) Handle(method string) (returnHandler, error) {
	h := self.handler(method)
	if h == nil {
		return nil, fmt.Errorf("allowed methods: %v", self.methods)
	}
	rh := func(r io.Reader) (io.ReadSeeker, error) {
		input, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		var arg interface{}
		if len(input) == 0 {
			input = []byte("{}")
		}
		if h.Type != nil {
			val := reflect.New(h.Type)
			req := val.Elem()
			req.Set(reflect.Zero(h.Type))
			err = json.Unmarshal(input, val.Interface())
			if err != nil {
				return nil, err
			}
			arg = req.Interface()
		}
		res, err := h.Handle(arg, self)
		if err != nil {
			return nil, err
		}
		resp, err := json.MarshalIndent(res, "", "  ")
		return bytes.NewReader(resp), nil
	}
	return rh, nil
}

func (self *ResourceController) Resource(id string) (Controller, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	c, ok := self.controllers[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}

func (self *ResourceController) SetResource(id string, c Controller) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if c == nil {
		delete(self.controllers, id)
	} else {
		self.controllers[id] = c
	}
}

func (self *ResourceController) DeleteResource(id string) {
	delete(self.controllers, id)
}

func RandomNodeId() *adapters.NodeId {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	pubkey := crypto.FromECDSAPub(&key.PublicKey)
	return adapters.NewNodeId(pubkey[1:])
}

func RandomNodeIds(n int) []*adapters.NodeId {
	var ids []*adapters.NodeId
	for i := 0; i < n; i++ {
		ids = append(ids, RandomNodeId())
	}
	return ids
}
