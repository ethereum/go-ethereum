// Copyright 2018 The go-ethereum Authors
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
			db.sysLock.Lock()
			db.sysHistory.ActiveMemory = append(db.sysHistory.ActiveMemory[1:], activeMemory)
			db.sysHistory.VirtualMemory = append(db.sysHistory.VirtualMemory[1:], virtualMemory)
			db.sysHistory.NetworkIngress = append(db.sysHistory.NetworkIngress[1:], networkIngress)
			db.sysHistory.NetworkEgress = append(db.sysHistory.NetworkEgress[1:], networkEgress)
			db.sysHistory.ProcessCPU = append(db.sysHistory.ProcessCPU[1:], processCPU)
			db.sysHistory.SystemCPU = append(db.sysHistory.SystemCPU[1:], systemCPU)
			db.sysHistory.DiskRead = append(db.sysHistory.DiskRead[1:], diskRead)
			db.sysHistory.DiskWrite = append(db.sysHistory.DiskWrite[1:], diskWrite)
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
