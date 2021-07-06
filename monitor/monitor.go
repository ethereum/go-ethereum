package monitor

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/log"
	"time"
)

var systemUsageMonitor SystemUsageMonitor

type SystemUsage struct {
	CurrentTime time.Time
	CpuData     map[string]float64
	MemData     map[string]float64
	IOData      map[string]float64
}

type SystemDurationUsage struct {
	DurationTime time.Duration
	CpuData      map[string]float64
	MemData      map[string]float64
	IOData       map[string]float64
}

func GetCurrentTime() time.Time {
	return time.Now()
}

type ISystemUsageMonitor interface {
	GetSystemUsage() map[string]float64
}

type SystemUsageMonitor struct {
	tool             Tool
	startSystemUsage SystemUsage
	endSystemUsage   SystemUsage
}

func NewSystemUsageMonitor() *SystemUsageMonitor {

	if &systemUsageMonitor != nil {
		log.Info("reuse SystemUsageMonitor")
		return &systemUsageMonitor
	} else {

		sum := SystemUsageMonitor{
			*NewTool(),
			SystemUsage{},
			SystemUsage{},
		}
		return &sum
	}
}

func (sum *SystemUsageMonitor) GetSystemCurrentUsage() *SystemUsage {
	return &SystemUsage{
		GetCurrentTime(),
		sum.tool.GetCpuData(),
		sum.tool.GetMemData(),
		sum.tool.GetIOData(),
	}
}

func getMapDataDiff(d1 map[string]float64, d2 map[string]float64) *map[string]float64 {
	diff := map[string]float64{}
	for k, v := range d1 {
		diff[k] = d2[k] - v
	}
	return &diff
}

func (sum *SystemUsageMonitor) Start() {
	sum.startSystemUsage = *sum.GetSystemCurrentUsage()
}

func (sum *SystemUsageMonitor) End() {
	sum.endSystemUsage = *sum.GetSystemCurrentUsage()
}

func (sum *SystemUsageMonitor) GetSystemDurationUsage() *SystemDurationUsage {
	return &SystemDurationUsage{
		DurationTime: sum.startSystemUsage.CurrentTime.Sub(GetCurrentTime()),
		CpuData:      *getMapDataDiff(sum.startSystemUsage.CpuData, sum.endSystemUsage.CpuData),
		MemData:      *getMapDataDiff(sum.startSystemUsage.CpuData, sum.endSystemUsage.MemData),
		IOData:       *getMapDataDiff(sum.startSystemUsage.CpuData, sum.endSystemUsage.IOData),
	}
}

func (sdu *SystemDurationUsage) ToString() string {
	_json, err := json.MarshalIndent(sdu, "", "\t")
	if err != nil {
		log.Error(err.Error())
		return "Can not convert systemDurationUsage to string"
	}
	return string(_json)
}

func (scu *SystemUsage) ToString() string {
	_json, err := json.MarshalIndent(scu, "", "\t")
	if err != nil {
		log.Error(err.Error())
		return "Can not convert systemUsage to string"
	}
	return string(_json)
}
