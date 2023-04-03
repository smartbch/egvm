package extension

import (
	"fmt"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

const (
	U256ScriptTemplate = `
		const a = U256(%v)
		a.ToString()
	`

	HexToU256ScriptTemplate = `
		const b = HexToU256('%v')
		b.ToString()
	`

	BufToU256ScriptTemplate = `
		let buffer = new ArrayBuffer(32); // 32 bytes
		let view = new Uint8Array(buffer);
		view[31] = %v
		const c = BufToU256(buffer)
		c.ToString()
	`
)

func setupJsFunc(vm *goja.Runtime) {
	vm.Set("U256", U256)
	vm.Set("HexToU256", HexToU256)
	vm.Set("BufToU256", BufToU256)
}

func TestCreateU256(t *testing.T) {
	vm := goja.New()
	setupJsFunc(vm)

	v1, err := vm.RunString(fmt.Sprintf(U256ScriptTemplate, 1))
	require.NoError(t, err)
	require.EqualValues(t, "0x1", v1.ToString())

	v2, err := vm.RunString(fmt.Sprintf(HexToU256ScriptTemplate, "0x2"))
	require.NoError(t, err)
	require.EqualValues(t, "0x2", v2.ToString())

	v3, err := vm.RunString(fmt.Sprintf(BufToU256ScriptTemplate, "0x3"))
	require.NoError(t, err)
	require.EqualValues(t, "0x3", v3.ToString())
}
