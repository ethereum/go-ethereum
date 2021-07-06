package monitor

import (
	"fmt"
	"testing"
)

func TestTool_GetCpuData(t *testing.T) {
	tools := NewTool()
	cpuData := tools.GetCpuData()
	fmt.Print(cpuData)
}
