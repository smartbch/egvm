package context

import (
	"crypto/sha256"
	"reflect"
	"runtime"

	"github.com/dop251/goja"
	"github.com/tyler-smith/go-bip32"

	"github.com/smartbch/pureauth/egvm-script/extension"
	"github.com/smartbch/pureauth/egvm-script/types"
	"github.com/smartbch/pureauth/keygrantor"
)

type EGVMContext struct {
	config         string
	inputBufLists  [][]byte
	state          []byte
	outputBufLists [][]byte
	certs          []string
	privKey        extension.Bip32Key
}

var EGVMCtx *EGVMContext

func SetContext(job *types.LambdaJob, keygrantorUrl string) {
	EGVMCtx.config = job.Config
	EGVMCtx.inputBufLists = job.Inputs
	EGVMCtx.state = job.State
	EGVMCtx.certs = job.Certs
	// use local rand key to replace keygrantor for dev and test on darwin
	if runtime.GOOS == "darwin" {
		seed, err := bip32.NewSeed()
		if err != nil {
			panic(err)
		}
		privKey, err := bip32.NewMasterKey(seed)
		if err != nil {
			panic(err)
		}
		EGVMCtx.privKey = extension.NewBip32Key(privKey)
	} else {
		scriptHash := sha256.Sum256([]byte(job.Script))
		privKey, err := keygrantor.GetKeyFromKeyGrantor(keygrantorUrl, scriptHash)
		if err != nil {
			panic(err) // comment for core logic test
		}
		EGVMCtx.privKey = extension.NewBip32Key(privKey)
	}
}

func SetContextInputs(inputs [][]byte) {
	EGVMCtx.inputBufLists = inputs
}

func ResetContext() {
	EGVMCtx.config = ""
	EGVMCtx.inputBufLists = nil
	EGVMCtx.outputBufLists = nil
	EGVMCtx.state = nil
}

func CollectResult() *types.LambdaResult {
	return &types.LambdaResult{
		Outputs: EGVMCtx.outputBufLists,
		State:   EGVMCtx.state,
	}
}

// ------- for js --------

func GetEGVMContext(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(EGVMCtx)
}

func (e *EGVMContext) GetConfig(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(e.config)
}

func (e *EGVMContext) SetConfig(cfg string) {
	e.config = cfg
}

func (e *EGVMContext) GetCerts(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(e.certs)
}

func (e *EGVMContext) GetState(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(vm.NewArrayBuffer(e.state))
}

func (e *EGVMContext) SetState(s goja.Value, vm *goja.Runtime) {
	switch s.Export().(type) {
	case goja.ArrayBuffer:
		e.state = s.Export().(goja.ArrayBuffer).Bytes()
	default:
		panic(vm.ToValue("param should be arraybuffer"))
	}
}

func (e *EGVMContext) GetInputs(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	var res []goja.ArrayBuffer
	for _, input := range e.inputBufLists {
		res = append(res, vm.NewArrayBuffer(input))
	}
	return vm.ToValue(res)
}

func (e *EGVMContext) SetOutputs(s goja.Value, vm *goja.Runtime) {
	switch t := s.Export().(type) {
	case []interface{}:
		outputBufLists := s.Export().([]interface{})
		for _, output := range outputBufLists {
			switch output.(type) {
			case goja.ArrayBuffer:
				e.outputBufLists = append(e.outputBufLists, output.(goja.ArrayBuffer).Bytes())
			default:
				panic(vm.ToValue("param not arraybuffer type"))
			}
		}
	case goja.ArrayBuffer:
		e.outputBufLists = append(e.outputBufLists, s.Export().(goja.ArrayBuffer).Bytes())
	default:
		panic(vm.ToValue("param not array type or arraybuffer, its:" + reflect.TypeOf(t).String()))
	}
}

func (e *EGVMContext) GetRootKey(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(e.privKey)
}
