package extension

import (
	"fmt"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

const (
	CPUIDScriptTemplate = `
		const cpuInfo = GetCPUID()
		const cpuInfoObj = JSON.parse(cpuInfo)
	`

	TSCScriptTemplate = `
		const n = 100
		const tscBz = GetTSC()
		const tsc = BufToU64BE(tscBz)

  		const startBz = GetTSCBenchStart()
		for (let i = 0; i < n; i++) {
			// code to evaluate
		}
		const endBz = GetTSCBenchEnd()

		const start = BufToU64BE(startBz)
		const end = BufToU64BE(endBz)
		const avg = (end - start - tsc) / n
	`
)

func setupGojaVmForCPU() *goja.Runtime {
	vm := goja.New()
	vm.Set("GetCPUID", GetCPUID)
	vm.Set("GetTSC", GetTSC)
	vm.Set("GetTSCBenchStart", GetTSCBenchStart)
	vm.Set("GetTSCBenchEnd", GetTSCBenchEnd)
	vm.Set("BufToU64BE", BufToU64BE)
	return vm
}

func TestGetCPUID(t *testing.T) {
	vm := setupGojaVmForCPU()
	_, err := vm.RunString(CPUIDScriptTemplate)
	require.NoError(t, err)

	cpuInfo := vm.Get("cpuInfo").Export().(string)
	require.NotEmpty(t, cpuInfo)
	fmt.Printf("cpuInfo: %s\n", cpuInfo)

	cpuInfoObj := vm.Get("cpuInfoObj").Export().(map[string]interface{})
	require.NotNil(t, cpuInfoObj)
	fmt.Printf("cpuInfoObj: %+v\n", cpuInfoObj)
}

func TestTSC(t *testing.T) {
	vm := setupGojaVmForCPU()
	_, err := vm.RunString(TSCScriptTemplate)
	require.NoError(t, err)

	avg := vm.Get("avg").Export().(float64)
	require.NotZero(t, avg)
	fmt.Printf("avg: %v\n", avg)
}
