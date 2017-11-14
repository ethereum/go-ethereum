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
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rcrowley/go-metrics"
	"golang.org/x/net/websocket"
)

const (
	memorySampleLimit  = 200 // Maximum number of memory data samples
	trafficSampleLimit = 200 // Maximum number of traffic data samples
)

var nextId uint32 // Next connection id

// Dashboard contains the dashboard internals.
type Dashboard struct {
	config *Config

	listener net.Listener
	conns    map[uint32]*client // Currently live websocket connections
	charts   charts             // The collected data samples to plot
	lock     sync.RWMutex       // Lock protecting the dashboard's internals

	quit chan chan error // Channel used for graceful exit
	wg   sync.WaitGroup
}

// message embraces the data samples of a client message.
type message struct {
	History *charts     `json:"history,omitempty"` // Past data samples
	Memory  *chartEntry `json:"memory,omitempty"`  // One memory sample
	Traffic *chartEntry `json:"traffic,omitempty"` // One traffic sample
	Log     string      `json:"log,omitempty"`     // One log
}

// client represents active websocket connection with a remote browser.
type client struct {
	conn   *websocket.Conn // Particular live websocket connection
	msg    chan message    // Message queue for the update messages
	logger log.Logger      // Logger for the particular live websocket connection
}

// charts contains the collected data samples.
type charts struct {
	Memory  []*chartEntry `json:"memorySamples,omitempty"`
	Traffic []*chartEntry `json:"trafficSamples,omitempty"`
}

// chartEntry represents one data sample
type chartEntry struct {
	Time  time.Time `json:"time,omitempty"`
	Value float64   `json:"value,omitempty"`
}

// New creates a new dashboard instance with the given configuration.
func New(config *Config) (*Dashboard, error) {
	return &Dashboard{
		conns:  make(map[uint32]*client),
		config: config,
		quit:   make(chan chan error),
	}, nil
}

// Protocols is a meaningless implementation of node.Service.
func (db *Dashboard) Protocols() []p2p.Protocol { return nil }

// APIs is a meaningless implementation of node.Service.
func (db *Dashboard) APIs() []rpc.API { return nil }

// Start implements node.Service, starting the data collection thread and the listening server of the dashboard.
func (db *Dashboard) Start(server *p2p.Server) error {
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
func (db *Dashboard) Stop() error {
	// Close the connection listener.
	var errs []error
	if err := db.listener.Close(); err != nil {
		errs = append(errs, err)
	}
	// Close the collectors.
	errc := make(chan error, 1)
	for i := 0; i < 2; i++ {
		db.quit <- errc
		if err := <-errc; err != nil {
			errs = append(errs, err)
		}
	}
	// Close the connections.
	db.lock.Lock()
	for _, c := range db.conns {
		if err := c.conn.Close(); err != nil {
			c.logger.Warn("Failed to close connection", "err", err)
		}
	}
	db.lock.Unlock()

	// Wait until every goroutine terminates.
	db.wg.Wait()
	log.Info("Dashboard stopped")

	var err error
	if len(errs) > 0 {
		err = fmt.Errorf("%v", errs)
	}

	return err
}

// webHandler handles all non-api requests, simply flattening and returning the dashboard website.
func (db *Dashboard) webHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("Request", "URL", r.URL)

	path := r.URL.String()
	if path == "/" {
		path = "/dashboard.html"
	}
	// If the path of the assets is manually set
	if db.config.Assets != "" {
		blob, err := ioutil.ReadFile(filepath.Join(db.config.Assets, path))
		if err != nil {
			log.Warn("Failed to read file", "path", path, "err", err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write(blob)
		return
	}
	blob, err := Asset(filepath.Join("public", path))
	if err != nil {
		log.Warn("Failed to load the asset", "path", path, "err", err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Write(blob)
}

// apiHandler handles requests for the dashboard.
func (db *Dashboard) apiHandler(conn *websocket.Conn) {
	id := atomic.AddUint32(&nextId, 1)
	client := &client{
		conn:   conn,
		msg:    make(chan message, 128),
		logger: log.New("id", id),
	}
	done := make(chan struct{}) // Buffered channel as sender may exit early

	// Start listening for messages to send.
	db.wg.Add(1)
	go func() {
		defer db.wg.Done()

		for {
			select {
			case <-done:
				return
			case msg := <-client.msg:
				if err := websocket.JSON.Send(client.conn, msg); err != nil {
					client.logger.Warn("Failed to send the message", "msg", msg, "err", err)
					client.conn.Close()
					return
				}
			}
		}
	}()
	// Send the past data.
	client.msg <- message{
		History: &db.charts,
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
			close(done)
			return
		}
		// Ignore all messages
	}
}

// collectData collects the required data to plot on the dashboard.
func (db *Dashboard) collectData() {
	defer db.wg.Done()

	for {
		select {
		case errc := <-db.quit:
			errc <- nil
			return
		case <-time.After(db.config.Refresh):
			inboundTraffic := metrics.DefaultRegistry.Get("p2p/InboundTraffic").(metrics.Meter).Rate1()
			memoryInUse := metrics.DefaultRegistry.Get("system/memory/inuse").(metrics.Meter).Rate1()
			now := time.Now()
			memory := &chartEntry{
				Time:  now,
				Value: memoryInUse,
			}
			traffic := &chartEntry{
				Time:  now,
				Value: inboundTraffic,
			}
			// Remove the first elements in case the samples' amount exceeds the limit.
			first := 0
			if len(db.charts.Memory) == memorySampleLimit {
				first = 1
			}
			db.charts.Memory = append(db.charts.Memory[first:], memory)
			first = 0
			if len(db.charts.Traffic) == trafficSampleLimit {
				first = 1
			}
			db.charts.Traffic = append(db.charts.Traffic[first:], traffic)

			db.sendToAll(&message{
				Memory:  memory,
				Traffic: traffic,
			})
		}
	}
}

// collectLogs collects and sends the logs to the active dashboards.
func (db *Dashboard) collectLogs() {
	defer db.wg.Done()

	// TODO (kurkomisi): log collection comes here.
	for {
		select {
		case errc := <-db.quit:
			errc <- nil
			return
		case <-time.After(db.config.Refresh / 2):
			db.sendToAll(&message{
				Log: "This is a fake log.",
			})
		}
	}
}

// sendToAll sends the given message to the active dashboards.
func (db *Dashboard) sendToAll(msg *message) {
	db.lock.Lock()
	for _, c := range db.conns {
		select {
		case c.msg <- *msg:
		default:
			c.conn.Close()
		}
	}
	db.lock.Unlock()
}
