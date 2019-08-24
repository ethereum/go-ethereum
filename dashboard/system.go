// Copyright 2019 The go-ethereum Authors
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

import (
	"runtime"
	"time"

	"github.com/elastic/gosigar"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
)

// meterCollector returns a function, which retrieves the count of a specific meter.
func meterCollector(name string) func() int64 {
	if meter := metrics.Get(name); meter != nil {
		m := meter.(metrics.Meter)
		return func() int64 {
			return m.Count()
		}
	}
	return func() int64 {
		return 0
	}
}

// collectSystemData gathers data about the system and sends it to the clients.
func (db *Dashboard) collectSystemData() {
	defer db.wg.Done()

	systemCPUUsage := gosigar.Cpu{}
	systemCPUUsage.Get()
	var (
		mem runtime.MemStats

		collectNetworkIngress = meterCollector(p2p.MetricsInboundTraffic)
		collectNetworkEgress  = meterCollector(p2p.MetricsOutboundTraffic)
		collectDiskRead       = meterCollector("eth/db/chaindata/disk/read")
		collectDiskWrite      = meterCollector("eth/db/chaindata/disk/write")

		prevNetworkIngress = collectNetworkIngress()
		prevNetworkEgress  = collectNetworkEgress()
		prevProcessCPUTime = getProcessCPUTime()
		prevSystemCPUUsage = systemCPUUsage
		prevDiskRead       = collectDiskRead()
		prevDiskWrite      = collectDiskWrite()

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
				curNetworkIngress = collectNetworkIngress()
				curNetworkEgress  = collectNetworkEgress()
				curProcessCPUTime = getProcessCPUTime()
				curSystemCPUUsage = systemCPUUsage
				curDiskRead       = collectDiskRead()
				curDiskWrite      = collectDiskWrite()

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

			runtime.ReadMemStats(&mem)
			activeMemory := &ChartEntry{
				Value: float64(mem.Alloc) / frequency,
			}
			virtualMemory := &ChartEntry{
				Value: float64(mem.Sys) / frequency,
			}
			networkIngress := &ChartEntry{
				Value: deltaNetworkIngress / frequency,
			}
			networkEgress := &ChartEntry{
				Value: deltaNetworkEgress / frequency,
			}
			processCPU := &ChartEntry{
				Value: deltaProcessCPUTime / frequency / numCPU * 100,
			}
			systemCPU := &ChartEntry{
				Value: float64(deltaSystemCPUUsage.Sys+deltaSystemCPUUsage.User) / frequency / numCPU,
			}
			diskRead := &ChartEntry{
				Value: float64(deltaDiskRead) / frequency,
			}
			diskWrite := &ChartEntry{
				Value: float64(deltaDiskWrite) / frequency,
			}
			db.sysLock.Lock()
			sys := db.history.System
			sys.ActiveMemory = append(sys.ActiveMemory[1:], activeMemory)
			sys.VirtualMemory = append(sys.VirtualMemory[1:], virtualMemory)
			sys.NetworkIngress = append(sys.NetworkIngress[1:], networkIngress)
			sys.NetworkEgress = append(sys.NetworkEgress[1:], networkEgress)
			sys.ProcessCPU = append(sys.ProcessCPU[1:], processCPU)
			sys.SystemCPU = append(sys.SystemCPU[1:], systemCPU)
			sys.DiskRead = append(sys.DiskRead[1:], diskRead)
			sys.DiskWrite = append(sys.DiskWrite[1:], diskWrite)
			db.sysLock.Unlock()

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
