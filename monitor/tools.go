package monitor

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"time"
)

type ITool interface {
	GetCpuData() map[string]float64
	GetMemData() map[string]float64
	GetIOData() map[string]float64
}

type Tool struct {
}

func NewTool() *Tool {
	return &Tool{}
}

func (t *Tool) GetCpuData() map[string]float64 {
	c2, _ := cpu.Percent(time.Duration(time.Second), false)
	return map[string]float64{
		"percent": float64(c2[0]),
	}
}

func (t *Tool) GetMemData() map[string]float64 {
	return map[string]float64{}
}

func (t *Tool) GetIOData() map[string]float64 {
	return map[string]float64{}
}

func main() {
	tool := NewTool()
	cpuData := tool.GetCpuData()
	memData := tool.GetMemData()
	ioData := tool.GetIOData()

	fmt.Print(cpuData)
	fmt.Print(memData)
	fmt.Print(ioData)

}
