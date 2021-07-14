package monitor

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
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
	c2, _ := cpu.Percent(time.Duration(time.Microsecond), false)
	return map[string]float64{
		"percent": float64(c2[0]),
	}
}

func (t *Tool) GetMemData() map[string]float64 {

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	var result = map[string]float64{}
	result["HeapObjects"] = float64(ms.HeapObjects)
	result["HeapAlloc"] = toMegaBytes(ms.HeapAlloc)
	result["TotalAlloc"] = toMegaBytes(ms.TotalAlloc)
	result["HeapSys"] = toMegaBytes(ms.HeapSys)
	result["HeapIdle"] = toMegaBytes(ms.HeapIdle)
	result["HeapReleased"] = toMegaBytes(ms.HeapReleased)
	result["RSS"] = toMegaBytes(ms.HeapIdle - ms.HeapReleased)

	//runtime.GC()
	return result
}

func toMegaBytes(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024
}

func (t *Tool) GetIOData() map[string]float64 {
	return map[string]float64{}
}

func (t *Tool) GetCpuDataByPID() string {
	pid := os.Getpid()
	prc := exec.Command("top", "-pid", strconv.Itoa(pid), "-l", "1", "-ncols", "3")

	out, err := prc.Output()
	if err != nil {
		panic(err)
	}
	words := strings.Split(string(out), " ")
	fmt.Println(words[len(words)-2])
	return string(out)
}
