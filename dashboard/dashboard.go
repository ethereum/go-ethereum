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

package dashboard

//go:generate go-bindata -nometadata -o assets.go -prefix assets -pkg dashboard assets/public/...

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rcrowley/go-metrics"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	processorSampleLimit = 200
	memorySampleLimit    = 200
	trafficSampleLimit   = 200
)

var (
	nextId uint32 // Next connection id
)

type dashboard struct {
	config *Config

	listener net.Listener
	conns    map[uint32]*client // Currently live websocket connections
	Metrics  *metricSamples     `json:"metrics,omitempty"`
	Stats    *status            `json:"stats,omitempty"`
	lock     sync.RWMutex       // Lock protecting the dashboard's internals

	closing chan chan error // Channel used for graceful exit
	wg      sync.WaitGroup
}

type client struct {
	conn   *websocket.Conn              // Particular live websocket connection
	msg    chan *map[string]interface{} // Message queue for the update messages
	logger log.Logger                   // Logger for the particular live websocket connection
}

type metricSamples struct {
	Processor []*data `json:"processor,omitempty"`
	Memory    []*data `json:"memory,omitempty"`
}

type data struct {
	Time  time.Time `json:"time,omitempty"`
	Value float64   `json:"value,omitempty"`
}

type status struct {
	Peers int `json:"peers,omitempty"`
	Block int `json:"block,omitempty"`
}

// New creates a new dashboard instance with the given configuration.
func New(config *Config) (*dashboard, error) {
	return &dashboard{
		conns:   make(map[uint32]*client),
		config:  config,
		Metrics: &metricSamples{},
		closing: make(chan chan error),
	}, nil
}

// Protocols is a meaningless implementation of node.Service.
func (db *dashboard) Protocols() []p2p.Protocol { return nil }

// APIs is a meaningless implementation of node.Service.
func (db *dashboard) APIs() []rpc.API { return nil }

// Start implements node.Service, starting the data collection thread and the listening server of the dashboard.
func (db *dashboard) Start(server *p2p.Server) error {
	db.wg.Add(2)
	go db.collectData()
	go db.collectLogs() // In case of removing this line change 2 back to 1 in wg.Add.

	http.HandleFunc("/", db.webHandler)
	http.Handle("/api", websocket.Handler(db.apiHandler))

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", db.config.Host, db.config.Port))
	if err != nil {
		return err
	}
	db.listener = listener

	go http.Serve(listener, nil)

	return nil
}

// Stop implements node.Service, stopping the data collection thread and the connection listener of the dashboard.
func (db *dashboard) Stop() error {
	var err error
	// Close the connection listener
	if err = db.listener.Close(); err != nil {
		log.Warn("Failed to close listener", "err", err)
	}

	errc := make(chan error)
	db.closing <- errc // collectData
	<-errc
	db.closing <- errc // collectLogs
	<-errc

	db.lock.Lock()
	for _, c := range db.conns {
		if err := c.conn.Close(); err != nil {
			c.logger.Warn("Failed to close connection", "err", err)
		}
	}
	db.lock.Unlock()

	db.wg.Wait()
	log.Info("Dashboard stopped")

	return err
}

func join(strings ...string) *bytes.Buffer {
	var buffer bytes.Buffer
	for _, s := range strings {
		buffer.WriteString(s)
	}
	return &buffer
}

// webHandler handles all non-api requests, simply flattening and returning the dashboard website.
func (db *dashboard) webHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("Request", "URL", r.URL)

	path := r.URL.String()
	if path == "/" {
		path = "/dashboard.html"
	}

	// If the path of the assets is manually set
	if db.config.Assets != "" {
		file, err := ioutil.ReadFile(join(db.config.Assets, path).String())
		if err != nil {
			log.Warn("Failed to read file", "err", err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write(file)
		return
	}

	webapp, err := Asset(join("public", path).String())
	if err != nil {
		log.Warn("Failed to load the asset", "path", path, "err", err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Write(webapp)
}

// apiHandler handles requests for the dashboard.
func (db *dashboard) apiHandler(conn *websocket.Conn) {
	id := atomic.AddUint32(&nextId, 1)
	client := &client{
		conn:   conn,
		msg:    make(chan *map[string]interface{}, 128),
		logger: log.New("id", id),
	}

	loss := make(chan bool, 1) // Buffered channel as sender may exit early

	// Start listening for messages to send.
	db.wg.Add(1)
	go func() {
		defer db.wg.Done()

		for {
			select {
			case <-loss:
				return
			case msg := <-client.msg:
				if err := websocket.JSON.Send(client.conn, msg); err != nil {
					client.logger.Warn("Failed to send the message", "msg", msg, "err", err)
					return
				}
			}
		}
	}()

	// Send the past data.
	client.msg <- &map[string]interface{}{
		"metrics": db.Metrics,
	}

	// Start tracking the connection and drop at connection loss.
	db.lock.Lock()
	db.conns[id] = client
	db.lock.Unlock()
	defer func() {
		db.lock.Lock()
		delete(db.conns, id)
		db.lock.Unlock()
	}()

	for {
		fail := []byte{}
		if _, err := conn.Read(fail); err != nil {
			loss <- true
			return
		}
		// Ignore all messages
	}
}

// collectData collects the required data to plot on the dashboard.
func (db *dashboard) collectData() {
	defer db.wg.Done()

	for {
		select {
		case errc := <-db.closing:
			errc <- nil
			return
		case <-time.After(db.config.Refresh):
			now := time.Now()
			traffic := metrics.DefaultRegistry.Get("p2p/InboundTraffic").(metrics.Meter).Rate1()
			memoryInUse := metrics.DefaultRegistry.Get("system/memory/inuse").(metrics.Meter).Rate1()
			traff := &data{
				Time:  now,
				Value: traffic,
			}
			memory := &data{
				Time:  now,
				Value: memoryInUse,
			}

			// TODO (kurkomisi): do not mix traffic with processor!
			db.update(traff, memory)
		}
	}
}

// collectLogs collects and sends the logs to the active dashboards.
func (db *dashboard) collectLogs() {
	defer db.wg.Done()

	// TODO (kurkomisi): log collection comes here.
	for {
		select {
		case errc := <-db.closing:
			errc <- nil
			return
		case <-time.After(db.config.Refresh):
			db.sendToAll(&map[string]interface{}{
				"log": "This is a fake log.",
			})
		}
	}
}

// update updates the dashboards through the live websocket connections.
func (db *dashboard) update(processor *data, memory *data) {
	// Remove the first elements in case the samples' amount exceeds the limit.
	first := 0
	if len(db.Metrics.Processor) == processorSampleLimit {
		first = 1
	}
	db.Metrics.Processor = append(db.Metrics.Processor[first:], processor)
	first = 0
	if len(db.Metrics.Memory) == memorySampleLimit {
		first = 1
	}
	db.Metrics.Memory = append(db.Metrics.Memory[first:], memory)

	db.sendToAll(&map[string]interface{}{
		"processor": processor,
		"memory":    memory,
	})
}

// Sends the given message to the active dashboards.
func (db *dashboard) sendToAll(msg *map[string]interface{}) {
	db.lock.Lock()
	for _, c := range db.conns {
		select {
		case c.msg <- msg:
		default:
			c.logger.Warn("Client message queue is full")
		}
	}
	db.lock.Unlock()
}
