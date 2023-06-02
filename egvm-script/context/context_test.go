package context

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"

	"github.com/smartbch/egvm/egvm-script/types"
)

func TestEGVMContextConfigRW(t *testing.T) {
	vm := goja.New()
	EGVMCtx = &EGVMContext{
		config: "a",
	}
	vm.Set("GetEGVMContext", GetEGVMContext)
	_, err := vm.RunString(`
	let ctx = GetEGVMContext() 
	let config = ctx.GetConfig()
	ctx.SetConfig("b")
`)
	require.Nil(t, err)
	c := vm.Get("config").Export().(string)
	require.Equal(t, c, "a")
	require.Equal(t, EGVMCtx.config, "b")
}

func TestEGVMContextStateRW(t *testing.T) {
	m := types.NewOrderedIntMap()
	m.Set("a", 1)
	m.Set("b", 2)

	vm := goja.New()
	s := types.SerializeMaps(goja.FunctionCall{
		This:      nil,
		Arguments: []goja.Value{vm.ToValue(m)},
	}, vm)
	EGVMCtx = &EGVMContext{
		state: s.Export().(goja.ArrayBuffer).Bytes(),
	}
	vm.Set("GetEGVMContext", GetEGVMContext)
	vm.Set("NewOrderedMapReader", types.NewOrderedMapReader)
	vm.Set("SerializeMaps", types.SerializeMaps)

	_, err := vm.RunString(`
	let EGVMCtx = GetEGVMContext() 
	let stateBz = EGVMCtx.GetState()
	let r = NewOrderedMapReader(stateBz)
	let m = r.Read(0)
	const [v, ok] = m.Get('a')
	const len = m.Len()
	m.Delete("a")
	m.Delete("b")
	const bz = SerializeMaps(m)
	EGVMCtx.SetState(bz)
`)
	require.NoError(t, err)
	require.Equal(t, int64(1), vm.Get("v").Export().(int64))
	require.Equal(t, 1+1 /* tag + map size*/, len(EGVMCtx.state))
}

func TestEGVMContextInputsR(t *testing.T) {
	m := types.NewOrderedIntMap()
	m.Set("a", 1)
	m.Set("b", 2)

	n := types.NewOrderedIntMap()
	n.Set("c", 3)
	n.Set("d", 4)
	vm := goja.New()
	im := types.SerializeMaps(goja.FunctionCall{
		This:      nil,
		Arguments: []goja.Value{vm.ToValue(m)},
	}, vm)
	in := types.SerializeMaps(goja.FunctionCall{
		This:      nil,
		Arguments: []goja.Value{vm.ToValue(n)},
	}, vm)
	EGVMCtx = &EGVMContext{
		inputBufLists: [][]byte{im.Export().(goja.ArrayBuffer).Bytes(), in.Export().(goja.ArrayBuffer).Bytes()},
	}
	vm.Set("GetEGVMContext", GetEGVMContext)
	vm.Set("NewOrderedMapReader", types.NewOrderedMapReader)
	vm.Set("SerializeMaps", types.SerializeMaps)

	_, err := vm.RunString(`
	let EGVMCtx = GetEGVMContext()
	let inputs = EGVMCtx.GetInputs()
	let l = inputs.length
	let r = NewOrderedMapReader(inputs[0])
	let m = r.Read(0)
	const [v, ok] = m.Get('a')
	const len = m.Len()
	let r1 = NewOrderedMapReader(inputs[1])
	let m1 = r1.Read(0)
	const [v1, ok1] = m1.Get('c')
`)
	require.Nil(t, err)
	require.Equal(t, int64(2), vm.Get("l").Export().(int64))
	require.Equal(t, int64(1), vm.Get("v").Export().(int64))
	require.Equal(t, int64(3), vm.Get("v1").Export().(int64))
}

func TestEGVMContextOutputsW(t *testing.T) {
	vm := goja.New()
	EGVMCtx = &EGVMContext{}
	vm.Set("GetEGVMContext", GetEGVMContext)
	vm.Set("NewOrderedMapReader", types.NewOrderedMapReader)
	vm.Set("SerializeMaps", types.SerializeMaps)

	_, err := vm.RunString(`
	let EGVMCtx = GetEGVMContext()
	let outs = new Array(3)
	outs[0] = new ArrayBuffer(1)
	outs[1] = new ArrayBuffer(2)
	outs[2] = new ArrayBuffer(3)
	
	EGVMCtx.SetOutputs(outs)
`)
	require.Nil(t, err)
	require.Equal(t, 3, len(EGVMCtx.outputBufLists))
	require.Equal(t, 1, len(EGVMCtx.outputBufLists[0]))
	require.Equal(t, 2, len(EGVMCtx.outputBufLists[1]))
	require.Equal(t, 3, len(EGVMCtx.outputBufLists[2]))
}

func TestEGVMContextCertsR(t *testing.T) {
	vm := goja.New()
	EGVMCtx = &EGVMContext{certs: []string{
		"abc", "edf",
	}}
	vm.Set("GetEGVMContext", GetEGVMContext)
	vm.Set("NewOrderedMapReader", types.NewOrderedMapReader)
	vm.Set("SerializeMaps", types.SerializeMaps)

	_, err := vm.RunString(`
	let EGVMCtx = GetEGVMContext()
	let certs = EGVMCtx.GetCerts()
`)
	require.Nil(t, err)
	certs := vm.Get("certs").Export().([]string)
	require.Equal(t, 2, len(certs))
	require.Equal(t, "abc", certs[0])
}

func TestEGVMContextRootKeyR(t *testing.T) {
	vm := goja.New()
	EGVMCtx = &EGVMContext{}
	SetContext(&types.LambdaJob{}, "")
	rootS := EGVMCtx.privKey.B58Serialize()
	vm.Set("GetEGVMContext", GetEGVMContext)
	vm.Set("NewOrderedMapReader", types.NewOrderedMapReader)
	vm.Set("SerializeMaps", types.SerializeMaps)

	_, err := vm.RunString(`
	let EGVMCtx = GetEGVMContext()
	let key = EGVMCtx.GetRootKey()
	let out = key.B58Serialize()
`)
	require.Nil(t, err)
	out := vm.Get("out").Export().(string)
	require.Equal(t, rootS, out)
}
