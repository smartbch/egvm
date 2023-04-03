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

	ToS256ScriptTemplate = `
		const d = U256(%v)
		const sd = d.ToS256()
		sd.ToString()
	`

	U256AddScriptTemplate = `
		const a = U256(%v)
		const b = U256(%v)
		const ab = a.Add(b)
		ab.ToString()
	`

	U256UnsafeAddScriptTemplate = `
		const c = U256(%v)
		const d = U256(%v)
		const cd = c.UnsafeAdd(d)
		cd.ToString()
	`

	U256AndScriptTemplate = `
		const a = HexToU256('%v')
		const b = HexToU256('%v')
		const ab = a.And(b)
		ab.ToString()
	`

	U256DivScriptTemplate = `
		const a = U256(%v)
		const b = U256(%v)
		const ab = a.Div(b)
		ab.ToString()
	`

	U256ModScriptTemplate = `
		const a = U256(%v)
		const b = U256(%v)
		const ab = a.Mod(b)
		ab.ToString()
	`

	U256DivModScriptTemplate = `
		const a = U256(%v)
		const b = U256(%v)
		const [z, m] = a.DivMod(b)
	`

	U256ExpScriptTemplate = `
		const a = U256(%v)
		const b = U256(%v)
		const ab = a.Exp(b)
		ab.ToString()
	`
)

func setupGojaVm() *goja.Runtime {
	vm := goja.New()

	vm.Set("U256", U256)
	vm.Set("HexToU256", HexToU256)
	vm.Set("BufToU256", BufToU256)
	return vm
}

func TestCreateU256(t *testing.T) {
	vm := setupGojaVm()

	// U256
	v1, err := vm.RunString(fmt.Sprintf(U256ScriptTemplate, 1))
	require.NoError(t, err)
	require.EqualValues(t, "0x1", v1.ToString())

	// HexToU256
	v2, err := vm.RunString(fmt.Sprintf(HexToU256ScriptTemplate, "0x2"))
	require.NoError(t, err)
	require.EqualValues(t, "0x2", v2.ToString())

	// BufToU256
	v3, err := vm.RunString(fmt.Sprintf(BufToU256ScriptTemplate, 3))
	require.NoError(t, err)
	require.EqualValues(t, "0x3", v3.ToString())
}

func TestToS256(t *testing.T) {
	// positive
	vm := setupGojaVm()
	v1, err := vm.RunString(fmt.Sprintf(ToS256ScriptTemplate, 100))
	require.NoError(t, err)
	require.EqualValues(t, "0x64", v1.ToString())

	// zero
	vm = setupGojaVm()
	v2, err := vm.RunString(fmt.Sprintf(ToS256ScriptTemplate, 0))
	require.NoError(t, err)
	require.EqualValues(t, "0x0", v2.ToString())
}

func TestU256Add(t *testing.T) {
	vm := setupGojaVm()

	v1, err := vm.RunString(fmt.Sprintf(U256AddScriptTemplate, 5, 8))
	require.NoError(t, err)
	require.EqualValues(t, "0xd", v1.ToString())

	v2, err := vm.RunString(fmt.Sprintf(U256UnsafeAddScriptTemplate, 5, 8))
	require.NoError(t, err)
	require.EqualValues(t, "0xd", v2.ToString())

	// overflow
	vm = setupGojaVm()
	v3, err := vm.RunString(fmt.Sprintf(U256UnsafeAddScriptTemplate, MAX_SAFE_INTEGER, MAX_SAFE_INTEGER))
	require.NoError(t, err)
	require.EqualValues(t, "0x3ffffffffffffe", v3.ToString())
}

func TestU256And(t *testing.T) {
	vm := setupGojaVm()

	v1, err := vm.RunString(fmt.Sprintf(U256AndScriptTemplate, "0xFF", "0xF0"))
	require.NoError(t, err)
	require.EqualValues(t, "0xf0", v1.ToString())
}

func TestU256DivAndMod(t *testing.T) {
	// Div
	vm := setupGojaVm()
	v1, err := vm.RunString(fmt.Sprintf(U256DivScriptTemplate, 10, 3))
	require.NoError(t, err)
	require.EqualValues(t, "0x3", v1.ToString())

	// Mod
	vm = setupGojaVm()
	v2, err := vm.RunString(fmt.Sprintf(U256ModScriptTemplate, 10, 3))
	require.NoError(t, err)
	require.EqualValues(t, "0x1", v2.ToString())

	// DivMod
	vm = setupGojaVm()
	_, err = vm.RunString(fmt.Sprintf(U256DivModScriptTemplate, 10, 3))
	require.NoError(t, err)

	z := vm.Get("z").Export().(Uint256)
	require.EqualValues(t, "0x3", z.x.String())
	m := vm.Get("m").Export().(Uint256)
	require.EqualValues(t, "0x1", m.x.String())
}

func TestU256Exp(t *testing.T) {
	vm := setupGojaVm()
	v1, err := vm.RunString(fmt.Sprintf(U256ExpScriptTemplate, 10, 2))
	require.NoError(t, err)
	require.EqualValues(t, "0x64", v1.ToString())
}
