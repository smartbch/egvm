package extension

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"github.com/dop251/goja"
	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/pureauth/egvm-script/utils"
)

func BufConcat(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	var data [][]byte
	totalLen := 0
	for _, arg := range f.Arguments {
		switch v := arg.Export().(type) {
		case goja.ArrayBuffer:
			data = append(data, v.Bytes())
			totalLen += len(v.Bytes())
		default:
			panic(vm.ToValue("Unsupported type for BufConcat"))
		}
	}

	result := make([]byte, 0, totalLen)
	for _, bz := range data {
		result = append(result, bz...)
	}
	return vm.ToValue(vm.NewArrayBuffer(result))
}

func B64ToBuf(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 1 {
		panic(utils.IncorrectArgumentCount)
	}
	str, ok := f.Arguments[0].Export().(string)
	if !ok {
		panic(goja.NewSymbol("The first argument must be string"))
	}
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		panic(goja.NewSymbol("error in B64ToBuf: " + err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(data))
}

func HexToBuf(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 1 {
		panic(utils.IncorrectArgumentCount)
	}
	str, ok := f.Arguments[0].Export().(string)
	if !ok {
		panic(goja.NewSymbol("The first argument must be string"))
	}

	data := gethcmn.FromHex(str)
	return vm.ToValue(vm.NewArrayBuffer(data))
}

func BufToB64(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	buf := utils.GetOneArrayBuffer(f)
	str := base64.StdEncoding.EncodeToString(buf)
	return vm.ToValue(str)
}

func BufToHex(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	buf := utils.GetOneArrayBuffer(f)
	str := hex.EncodeToString(buf)
	return vm.ToValue(str)
}

func BufEqual(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	a, b := utils.GetTwoArrayBuffers(f)
	return vm.ToValue(bytes.Equal(a, b))
}

func BufCompare(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	a, b := utils.GetTwoArrayBuffers(f)
	return vm.ToValue(bytes.Compare(a, b))
}

func BufReverse(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	a := utils.GetOneArrayBuffer(f)
	b := make([]byte, 0, len(a))
	for i := range a {
		b = append(b, a[len(a)-1-i])
	}
	return vm.ToValue(vm.NewArrayBuffer(b))
}

func BufToU64BE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	bz := utils.GetOneArrayBuffer(f)
	return vm.ToValue(binary.BigEndian.Uint64(bz))
}

func BufToU64LE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	bz := utils.GetOneArrayBuffer(f)
	return vm.ToValue(binary.LittleEndian.Uint64(bz))
}

func BufToU32BE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	bz := utils.GetOneArrayBuffer(f)
	return vm.ToValue(binary.BigEndian.Uint32(bz))
}

func BufToU32LE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	bz := utils.GetOneArrayBuffer(f)
	return vm.ToValue(binary.LittleEndian.Uint32(bz))
}

func U64ToBufBE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u64 := utils.GetOneUint64(f)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], u64)
	return vm.ToValue(vm.NewArrayBuffer(buf[:]))
}

func U64ToBufLE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u64 := utils.GetOneUint64(f)
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], u64)
	return vm.ToValue(vm.NewArrayBuffer(buf[:]))
}

func U32ToBufBE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u64 := utils.GetOneUint64(f)
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(u64))
	return vm.ToValue(vm.NewArrayBuffer(buf[:]))
}

func U32ToBufLE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u64 := utils.GetOneUint64(f)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(u64))
	return vm.ToValue(vm.NewArrayBuffer(buf[:]))
}
