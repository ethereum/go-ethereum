package monitor

import (
	"testing"
)

func BenchmarkTool_GetCpuData(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool := NewTool()
		tool.GetCpuData()
	}
}

func TestTool_GetCpuDataByPID(t *testing.T) {

	tool := NewTool()
	tool.GetCpuDataByPID()

}

func BenchmarkTool_GetCpuDataByPID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool := NewTool()
		tool.GetCpuDataByPID()
		//fmt.Println(cpuDatauData)
	}
}

func BenchmarkTool_GetMemData(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool := NewTool()
		tool.GetMemData()
	}
}

func BenchmarkTool_GetIOData(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool := NewTool()
		tool.GetIOData()
	}
}
