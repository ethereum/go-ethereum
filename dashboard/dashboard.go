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

//go:generate go-bindata -nometadata -o assets.go -prefix assets -pkg dashboard assets/...

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rcrowley/go-metrics"
	"golang.org/x/net/websocket"
	"html/template"
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
	nextId uint32 = 0 // Next connection id
)

type dashboard struct {
	config *Config

	listener net.Listener
	conns    []*client // Currently live websocket connections

	Metrics *metricSamples `json:"metrics,omitempty"`
	Stats   *status        `json:"stats,omitempty"`

	lock sync.RWMutex // Lock protecting the dashboard's internals

	quit chan struct{} // Channel used for graceful exit
}

type client struct {
	conn   *websocket.Conn // Particular live websocket connection
	logger log.Logger      // Logger for the particular live websocket connection
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

// New creates a new dashboard instance with the given configuration
func New(config *Config) (*dashboard, error) {
	//log.Trace("NewDashboard() called")

	return &dashboard{
		config:  config,
		Metrics: &metricSamples{},
		quit:    make(chan struct{}),
	}, nil
}

// Protocols is a meaningless implementation of node.Service
func (db *dashboard) Protocols() []p2p.Protocol { return nil }

// APIs is a meaningless implementation of node.Service
func (db *dashboard) APIs() []rpc.API { return nil }

// Start implements node.Service, starting the data collection thread and the listening server of the dashboard
func (db *dashboard) Start(server *p2p.Server) error {
	//log.Trace("Start() called", "config", db.config)

	go db.collectData()

	http.HandleFunc("/", db.webHandler)
	http.Handle("/api", websocket.Handler(db.apiHandler))

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", db.config.Host, db.config.Port))
	if err != nil {
		return err
	}
	db.listener = listener

	go func() {
		//log.Trace("Starting server...")

		if err := http.Serve(listener, nil); err != nil {
			log.Warn("Server failed", "err", err)
		}
	}()

	return nil
}

// Stop implements node.Service, stopping the data collection thread and the connection listener of the dashboard
func (db *dashboard) Stop() error {
	//log.Trace("Terminating dashboard...")

	db.lock.Lock()
	defer db.lock.Unlock()

	var err error
	// Close the connection listener
	if err = db.listener.Close(); err != nil {
		log.Warn("Failed to close listener", "err", err)
	}

	close(db.quit) // Notifies collectData and apiHandler

	for _, c := range db.conns {
		if err := c.conn.Close(); err != nil {
			c.logger.Warn("Failed to close connection", "err", err)
		}
	}
	db.conns = db.conns[:0]

	return err
}

// webHandler handles all non-api requests, simply flattening and returning the dashboard website.
func (db *dashboard) webHandler(w http.ResponseWriter, r *http.Request) {
	//log.Trace("webHandler() called", "r.URL", r.URL)
	log.Info("Request", "URL", r.URL)

	path := r.URL.String()

	// If the path of the assets is manually set
	if db.config.Assets != "" {
		// Create the filename for ReadFile
		var buffer bytes.Buffer
		buffer.WriteString(db.config.Assets)
		switch path {
		case "/":
			buffer.WriteString("/dashboard.html")
		default:
			buffer.WriteString(path)
		}

		file, err := ioutil.ReadFile(buffer.String())
		if err != nil {
			// TODO (kurkomisi): Warn or Crit?
			log.Warn("Failed to read file", "err", err)
			// TODO (kurkomisi): Should I inform the client?
			return
		}
		w.Write(file)
		return
	}

	switch path {
	case "/":
		index, err := Asset("dashboard.html")
		if err != nil {
			log.Warn("Failed to load the index", "err", err)
			return
		}
		w.Write(index)
	case "/js/handlers.js":
		tmpl, err := Asset("js/handlers.js")
		if err != nil {
			log.Warn("Failed to load the asset", "path", path, "err", err)
			return
		}
		handlers := new(bytes.Buffer)
		// TODO (kurkomisi): Save the generated template to avoid the repeated generation?
		// set the sample limits for the client
		if err = template.Must(template.New("").Parse(string(tmpl))).Execute(handlers, map[string]interface{}{
			"processorSampleLimit": processorSampleLimit,
			"memorySampleLimit":    memorySampleLimit,
			"trafficSampleLimit":   trafficSampleLimit,
		}); err != nil {
			log.Warn("Failed to render the dashboard handlers template", "err", err)
			return
		}
		w.Write(handlers.Bytes())
	default:
		website, err := Asset(path[1:])
		if err != nil {
			log.Warn("Failed to load the asset", "path", path, "err", err)
			return
		}
		w.Write(website)
	}
}

// apiHandler handles requests for dashboard
func (db *dashboard) apiHandler(conn *websocket.Conn) {
	//log.Trace("apiHandler() called")

	client := &client{
		conn:   conn,
		logger: log.New("id", atomic.AddUint32(&nextId, 1)),
	}

	// Start tracking the connection and drop at connection loss
	db.lock.Lock()
	db.conns = append(db.conns, client)
	db.lock.Unlock()

	db.sendHistory(client)

	closed := make(chan bool)

	go func() {
		select {
		case <-db.quit:
			//client.logger.Trace("apiHandler closed")
		case <-closed:
			//client.logger.Trace("Connection interrupted")
			db.lock.Lock()
			for i, c := range db.conns {
				if c.conn == client.conn {
					db.conns = append(db.conns[:i], db.conns[i+1:]...)
					break
				}
			}
			db.lock.Unlock()
		}
	}()

	for {
		fail := []byte{}
		if _, err := conn.Read(fail); err != nil {
			closed <- true
			return
		}
		// Ignore all messages
	}
}

// collectData collects the required data to plot on the dashboard
func (db *dashboard) collectData() {
	//log.Trace("collectData() called")

	for {
		select {
		case <-db.quit:
			//log.Trace("collectData closed")
			return
		case <-time.After(db.config.Refresh):

			now := time.Now()
			traffic := metrics.DefaultRegistry.Get("p2p/InboundTraffic").(metrics.Meter).Rate1()
			//if traffic != 0 {
			//	traffic = math.Log(traffic)
			//}
			memoryInUse := metrics.DefaultRegistry.Get("system/memory/inuse").(metrics.Meter).Rate1()
			//if memoryInuse != 0 {
			//	memoryInuse = math.Log(memoryInuse)
			//}
			traff := &data{
				Time:  now,
				Value: traffic,
			}
			memory := &data{
				Time:  now,
				Value: memoryInUse,
			}

			// TODO (kurkomisi): do I need to ensure the correct order?
			// TODO (kurkomisi): do not mix traffic with processor!
			go db.update(traff, memory)
		}
	}
}

// update updates the dashboards through the live websocket connections
func (db *dashboard) update(processor *data, memory *data) {
	//log.Trace("update() called")

	db.lock.Lock()
	defer db.lock.Unlock()

	// if the samples' # exceeds the limit, just remove the first element
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

	for _, c := range db.conns {
		//c.logger.Trace("Updating dashboard...")

		msg := &map[string]interface{}{
			"processor": processor,
			"memory":    memory,
		}
		if err := websocket.JSON.Send(c.conn, msg); err != nil {
			c.logger.Warn("Failed to update dashboard", "msg", msg, "err", err)
		}
	}

}

// sendHistory sends the past data through a newly registered websocket connection
func (db *dashboard) sendHistory(c *client) {
	//c.logger.Trace("Sending history...")

	msg := &map[string]interface{}{
		"metrics": db.Metrics,
	}
	if err := websocket.JSON.Send(c.conn, msg); err != nil {
		c.logger.Warn("Failed to send history", "err", err)
	}
}
