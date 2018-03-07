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

//go:generate yarn --cwd ./assets install
//go:generate yarn --cwd ./assets build
//go:generate go-bindata -nometadata -o assets.go -prefix assets -nocompress -pkg dashboard assets/index.html assets/bundle.js
//go:generate sh -c "sed 's#var _bundleJs#//nolint:misspell\\\n&#' assets.go > assets.go.tmp && mv assets.go.tmp assets.go"
//go:generate sh -c "sed 's#var _indexHtml#//nolint:misspell\\\n&#' assets.go > assets.go.tmp && mv assets.go.tmp assets.go"
//go:generate gofmt -w -s assets.go

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/gosigar"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/websocket"
)

const (
	activeMemorySampleLimit   = 200 // Maximum number of active memory data samples
	virtualMemorySampleLimit  = 200 // Maximum number of virtual memory data samples
	networkIngressSampleLimit = 200 // Maximum number of network ingress data samples
	networkEgressSampleLimit  = 200 // Maximum number of network egress data samples
	processCPUSampleLimit     = 200 // Maximum number of process cpu data samples
	systemCPUSampleLimit      = 200 // Maximum number of system cpu data samples
	diskReadSampleLimit       = 200 // Maximum number of disk read data samples
	diskWriteSampleLimit      = 200 // Maximum number of disk write data samples
)

var nextID uint32 // Next connection id

// Dashboard contains the dashboard internals.
type Dashboard struct {
	config *Config

	listener net.Listener
	conns    map[uint32]*client // Currently live websocket connections
	charts   *SystemMessage
	commit   string
	lock     sync.RWMutex // Lock protecting the dashboard's internals

	quit chan chan error // Channel used for graceful exit
	wg   sync.WaitGroup
}

// client represents active websocket connection with a remote browser.
type client struct {
	conn   *websocket.Conn // Particular live websocket connection
	msg    chan Message    // Message queue for the update messages
	logger log.Logger      // Logger for the particular live websocket connection
}

// New creates a new dashboard instance with the given configuration.
func New(config *Config, commit string) (*Dashboard, error) {
	now := time.Now()
	db := &Dashboard{
		conns:  make(map[uint32]*client),
		config: config,
		quit:   make(chan chan error),
		charts: &SystemMessage{
			ActiveMemory:   emptyChartEntries(now, activeMemorySampleLimit, config.Refresh),
			VirtualMemory:  emptyChartEntries(now, virtualMemorySampleLimit, config.Refresh),
			NetworkIngress: emptyChartEntries(now, networkIngressSampleLimit, config.Refresh),
			NetworkEgress:  emptyChartEntries(now, networkEgressSampleLimit, config.Refresh),
			ProcessCPU:     emptyChartEntries(now, processCPUSampleLimit, config.Refresh),
			SystemCPU:      emptyChartEntries(now, systemCPUSampleLimit, config.Refresh),
			DiskRead:       emptyChartEntries(now, diskReadSampleLimit, config.Refresh),
			DiskWrite:      emptyChartEntries(now, diskWriteSampleLimit, config.Refresh),
		},
		commit: commit,
	}
	return db, nil
}

// emptyChartEntries returns a ChartEntry array containing limit number of empty samples.
func emptyChartEntries(t time.Time, limit int, refresh time.Duration) ChartEntries {
	ce := make(ChartEntries, limit)
	for i := 0; i < limit; i++ {
		ce[i] = &ChartEntry{
			Time: t.Add(-time.Duration(i) * refresh),
		}
	}
	return ce
}

// Protocols is a meaningless implementation of node.Service.
func (db *Dashboard) Protocols() []p2p.Protocol { return nil }

// APIs is a meaningless implementation of node.Service.
func (db *Dashboard) APIs() []rpc.API { return nil }

// Start implements node.Service, starting the data collection thread and the listening server of the dashboard.
func (db *Dashboard) Start(server *p2p.Server) error {
	log.Info("Starting dashboard")

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
		path = "/index.html"
	}
	blob, err := Asset(path[1:])
	if err != nil {
		log.Warn("Failed to load the asset", "path", path, "err", err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Write(blob)
}

// apiHandler handles requests for the dashboard.
func (db *Dashboard) apiHandler(conn *websocket.Conn) {
	id := atomic.AddUint32(&nextID, 1)
	client := &client{
		conn:   conn,
		msg:    make(chan Message, 128),
		logger: log.New("id", id),
	}
	done := make(chan struct{})

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

	versionMeta := ""
	if len(params.VersionMeta) > 0 {
		versionMeta = fmt.Sprintf(" (%s)", params.VersionMeta)
	}
	// Send the past data.
	client.msg <- Message{
		General: &GeneralMessage{
			Version: fmt.Sprintf("v%d.%d.%d%s", params.VersionMajor, params.VersionMinor, params.VersionPatch, versionMeta),
			Commit:  db.commit,
		},
		System: &SystemMessage{
			ActiveMemory:   db.charts.ActiveMemory,
			VirtualMemory:  db.charts.VirtualMemory,
			NetworkIngress: db.charts.NetworkIngress,
			NetworkEgress:  db.charts.NetworkEgress,
			ProcessCPU:     db.charts.ProcessCPU,
			SystemCPU:      db.charts.SystemCPU,
			DiskRead:       db.charts.DiskRead,
			DiskWrite:      db.charts.DiskWrite,
		},
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
	systemCPUUsage := gosigar.Cpu{}
	systemCPUUsage.Get()
	var (
		mem runtime.MemStats

		prevNetworkIngress = metrics.DefaultRegistry.Get("p2p/InboundTraffic").(metrics.Meter).Count()
		prevNetworkEgress  = metrics.DefaultRegistry.Get("p2p/OutboundTraffic").(metrics.Meter).Count()
		prevProcessCPUTime = getProcessCPUTime()
		prevSystemCPUUsage = systemCPUUsage
		prevDiskRead       = metrics.DefaultRegistry.Get("eth/db/chaindata/compact/input").(metrics.Meter).Count()
		prevDiskWrite      = metrics.DefaultRegistry.Get("eth/db/chaindata/compact/output").(metrics.Meter).Count()

		frequency = float64(db.config.Refresh / time.Second)
		numCPU    = float64(runtime.NumCPU())
	)

	for {
		select {
		case errc := <-db.quit:
			errc <- nil
			return
		case <-time.After(db.config.Refresh):
			systemCPUUsage.Get()
			var (
				curNetworkIngress = metrics.DefaultRegistry.Get("p2p/InboundTraffic").(metrics.Meter).Count()
				curNetworkEgress  = metrics.DefaultRegistry.Get("p2p/OutboundTraffic").(metrics.Meter).Count()
				curProcessCPUTime = getProcessCPUTime()
				curSystemCPUUsage = systemCPUUsage
				curDiskRead       = metrics.DefaultRegistry.Get("eth/db/chaindata/compact/input").(metrics.Meter).Count()
				curDiskWrite      = metrics.DefaultRegistry.Get("eth/db/chaindata/compact/output").(metrics.Meter).Count()

				deltaNetworkIngress = float64(curNetworkIngress - prevNetworkIngress)
				deltaNetworkEgress  = float64(curNetworkEgress - prevNetworkEgress)
				deltaProcessCPUTime = curProcessCPUTime - prevProcessCPUTime
				deltaSystemCPUUsage = curSystemCPUUsage.Delta(prevSystemCPUUsage)
				deltaDiskRead       = curDiskRead - prevDiskRead
				deltaDiskWrite      = curDiskWrite - prevDiskWrite
			)
			prevNetworkIngress = curNetworkIngress
			prevNetworkEgress = curNetworkEgress
			prevProcessCPUTime = curProcessCPUTime
			prevSystemCPUUsage = curSystemCPUUsage
			prevDiskRead = curDiskRead
			prevDiskWrite = curDiskWrite

			now := time.Now()

			runtime.ReadMemStats(&mem)
			activeMemory := &ChartEntry{
				Time:  now,
				Value: float64(mem.Alloc) / frequency,
			}
			virtualMemory := &ChartEntry{
				Time:  now,
				Value: float64(mem.Sys) / frequency,
			}
			networkIngress := &ChartEntry{
				Time:  now,
				Value: deltaNetworkIngress / frequency,
			}
			networkEgress := &ChartEntry{
				Time:  now,
				Value: deltaNetworkEgress / frequency,
			}
			processCPU := &ChartEntry{
				Time:  now,
				Value: deltaProcessCPUTime / frequency / numCPU * 100,
			}
			systemCPU := &ChartEntry{
				Time:  now,
				Value: float64(deltaSystemCPUUsage.Sys+deltaSystemCPUUsage.User) / frequency / numCPU,
			}
			diskRead := &ChartEntry{
				Time:  now,
				Value: float64(deltaDiskRead) / frequency,
			}
			diskWrite := &ChartEntry{
				Time:  now,
				Value: float64(deltaDiskWrite) / frequency,
			}
			db.charts.ActiveMemory = append(db.charts.ActiveMemory[1:], activeMemory)
			db.charts.VirtualMemory = append(db.charts.VirtualMemory[1:], virtualMemory)
			db.charts.NetworkIngress = append(db.charts.NetworkIngress[1:], networkIngress)
			db.charts.NetworkEgress = append(db.charts.NetworkEgress[1:], networkEgress)
			db.charts.ProcessCPU = append(db.charts.ProcessCPU[1:], processCPU)
			db.charts.SystemCPU = append(db.charts.SystemCPU[1:], systemCPU)
			db.charts.DiskRead = append(db.charts.DiskRead[1:], diskRead)
			db.charts.DiskWrite = append(db.charts.DiskRead[1:], diskWrite)

			db.sendToAll(&Message{
				System: &SystemMessage{
					ActiveMemory:   ChartEntries{activeMemory},
					VirtualMemory:  ChartEntries{virtualMemory},
					NetworkIngress: ChartEntries{networkIngress},
					NetworkEgress:  ChartEntries{networkEgress},
					ProcessCPU:     ChartEntries{processCPU},
					SystemCPU:      ChartEntries{systemCPU},
					DiskRead:       ChartEntries{diskRead},
					DiskWrite:      ChartEntries{diskWrite},
				},
			})
		}
	}
}

// collectLogs collects and sends the logs to the active dashboards.
func (db *Dashboard) collectLogs() {
	defer db.wg.Done()

	id := 1
	// TODO (kurkomisi): log collection comes here.
	for {
		select {
		case errc := <-db.quit:
			errc <- nil
			return
		case <-time.After(db.config.Refresh / 2):
			db.sendToAll(&Message{
				Logs: &LogsMessage{
					Log: []string{fmt.Sprintf("%-4d: This is a fake log.", id)},
				},
			})
			id++
		}
	}
}

// sendToAll sends the given message to the active dashboards.
func (db *Dashboard) sendToAll(msg *Message) {
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
