package request

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

const (
	AttestScriptTemplate = `
		const [ok, reason] = AttestEnclaveServer('https://elfincdn111.paralinker.io')
	`
)

func setupGojaVmForEnclave() *goja.Runtime {
	vm := goja.New()
	vm.Set("AttestEnclaveServer", AttestEnclaveServer)
	return vm
}

func TestAttestEnclaveServer(t *testing.T) {
	vm := setupGojaVmForEnclave()
	_, err := vm.RunString(AttestScriptTemplate)
	require.NoError(t, err)

	ok := vm.Get("ok").Export().(bool)
	reason := vm.Get("reason").Export().(string)
	require.True(t, ok)
	require.Empty(t, reason)
}
