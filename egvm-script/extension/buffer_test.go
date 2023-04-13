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
		const b64Str = 'YWJjZDEyMzQ='
		const bz = B64ToBuf(b64Str)
	`

	HexToBufScriptTemplate = `
		const hex = '0xff11'
		const bz = HexToBuf(hex)
	`

	BufToB64ScriptTemplate = `
		const buffer1 = new Uint8Array([97, 98, 99, 100, 49, 50, 51, 52]).buffer
		const b64Str = BufToB64(buffer1)
	`

	BufToHexScriptTemplate = `
		const buffer1 = new Uint8Array([255, 17]).buffer
		const hex = BufToHex(buffer1)
	`

	BufEqualAndCompareScriptTemplate = `
		const buffer1 = new Uint8Array([1, 2, 3, 4]).buffer
		const buffer2 = new Uint8Array([1, 2, 3, 4]).buffer
		const buffer3 = new Uint8Array([1, 2, 3, 5]).buffer
		const buffer4 = new Uint8Array([1, 2, 3, 3]).buffer

		const v1 = BufEqual(buffer1, buffer2)
		const v2 = BufCompare(buffer1, buffer2)
		const v3 = BufCompare(buffer1, buffer3)
		const v4 = BufCompare(buffer1, buffer4)
	`

	BufReverseScriptTemplate = `
		const buffer1 = new Uint8Array([1, 2, 3, 4]).buffer
		const reverseBz = BufReverse(buffer1)
	`

	BufToUintScriptTemplate = `
		const buffer1 = new Uint8Array([1, 2, 3, 4, 8, 7, 6, 5]).buffer
		const u64be = BufToU64BE(buffer1)
		const u64le = BufToU64LE(buffer1)
	`
)

func setupGojaVmForBuffer() *goja.Runtime {
	vm := goja.New()
	vm.Set("BufConcat", BufConcat)
	vm.Set("B64ToBuf", B64ToBuf)
	vm.Set("HexToBuf", HexToBuf)
	vm.Set("BufToB64", BufToB64)
	vm.Set("BufToHex", BufToHex)
	vm.Set("BufEqual", BufEqual)
	vm.Set("BufCompare", BufCompare)
	vm.Set("BufReverse", BufReverse)

	vm.Set("BufToU64BE", BufToU64BE)
	vm.Set("BufToU64LE", BufToU64LE)
	vm.Set("BufToU32BE", BufToU32BE)
	vm.Set("BufToU32LE", BufToU32LE)
	vm.Set("U64ToBufBE", U64ToBufBE)
	vm.Set("U64ToBufLE", U64ToBufLE)
	vm.Set("U32ToBufBE", U32ToBufBE)
	vm.Set("U32ToBufLE", U32ToBufLE)
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

func TestBufToB64(t *testing.T) {
	vm := setupGojaVmForBuffer()
	_, err := vm.RunString(BufToB64ScriptTemplate)
	require.NoError(t, err)

	b64Str := vm.Get("b64Str").Export().(string)
	require.EqualValues(t, "YWJjZDEyMzQ=", b64Str)
}

func TestBufToHex(t *testing.T) {
	vm := setupGojaVmForBuffer()
	_, err := vm.RunString(BufToHexScriptTemplate)
	require.NoError(t, err)

	hex := vm.Get("hex").Export().(string)
	require.EqualValues(t, "ff11", hex)
}

func TestBufEqualAndCompare(t *testing.T) {
	vm := setupGojaVmForBuffer()
	_, err := vm.RunString(BufEqualAndCompareScriptTemplate)
	require.NoError(t, err)

	v1 := vm.Get("v1").Export().(bool)
	v2 := vm.Get("v2").Export().(int64)
	v3 := vm.Get("v3").Export().(int64)
	v4 := vm.Get("v4").Export().(int64)
	require.True(t, v1)
	require.EqualValues(t, 0, v2)
	require.EqualValues(t, -1, v3)
	require.EqualValues(t, 1, v4)
}

func TestBufReverse(t *testing.T) {
	vm := setupGojaVmForBuffer()
	_, err := vm.RunString(BufReverseScriptTemplate)
	require.NoError(t, err)

	reverseBz := vm.Get("reverseBz").Export().(goja.ArrayBuffer)
	reverseBzHex := gethcmn.Bytes2Hex(reverseBz.Bytes())
	require.EqualValues(t, "04030201", reverseBzHex)
}

func TestBufToUint(t *testing.T) {
	vm := setupGojaVmForBuffer()
	_, err := vm.RunString(BufToUintScriptTemplate)
	require.NoError(t, err)

	u64be := vm.Get("u64be").Export().(float64)
	u64le := vm.Get("u64le").Export().(float64)
	require.EqualValues(t, 72623859840779781, u64be)
	require.EqualValues(t, 361984551058342401, u64le)
}
