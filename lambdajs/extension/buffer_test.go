package extension

import (
	"testing"

	"github.com/dop251/goja"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const (
	BufConcatScriptTemplate = `
		let buffer1 = new ArrayBuffer(8); // 8 bytes
		let buffer2 = new ArrayBuffer(8); // 8 bytes
		let buffer3 = new ArrayBuffer(8); // 8 bytes
		let buffer4 = new ArrayBuffer(8); // 8 bytes
		let view1 = new Uint8Array(buffer1);
		view1[7] = 1
		let view2 = new Uint8Array(buffer2);
		view2[7] = 2
		let view3 = new Uint8Array(buffer3);
		view3[7] = 3
		let view4 = new Uint8Array(buffer4);
		view4[7] = 4

		const bz = BufConcat(buffer1, buffer2, buffer3, buffer4)
	`

	B64ToBufScriptTemplate = `
		const base64Str = 'YWJjZDEyMzQ='
		const bz = B64ToBuf(base64Str)
	`

	HexToBufScriptTemplate = `
		const hex = '0xff11'
		const bz = HexToBuf(hex)
	`
)

func setupGojaVmForBuffer() *goja.Runtime {
	vm := goja.New()
	vm.Set("BufConcat", BufConcat)
	vm.Set("B64ToBuf", B64ToBuf)
	vm.Set("HexToBuf", HexToBuf)
	return vm
}

func TestBufConcat(t *testing.T) {
	vm := setupGojaVmForBuffer()
	_, err := vm.RunString(BufConcatScriptTemplate)
	require.NoError(t, err)

	bz := vm.Get("bz").Export().(goja.ArrayBuffer)
	bzHex := gethcmn.Bytes2Hex(bz.Bytes())
	require.EqualValues(t, "0000000000000001000000000000000200000000000000030000000000000004", bzHex)
}

func TestB64ToBuf(t *testing.T) {
	vm := setupGojaVmForBuffer()
	_, err := vm.RunString(B64ToBufScriptTemplate)
	require.NoError(t, err)

	bz := vm.Get("bz").Export().(goja.ArrayBuffer)
	bzHex := gethcmn.Bytes2Hex(bz.Bytes())
	require.EqualValues(t, "6162636431323334", bzHex)
}

func TestHexToBuf(t *testing.T) {
	vm := setupGojaVmForBuffer()
	_, err := vm.RunString(HexToBufScriptTemplate)
	require.NoError(t, err)

	bz := vm.Get("bz").Export().(goja.ArrayBuffer)
	bzHex := gethcmn.Bytes2Hex(bz.Bytes())
	require.EqualValues(t, "ff11", bzHex)
}
