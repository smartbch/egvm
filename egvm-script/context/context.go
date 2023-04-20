package context

import (
	"github.com/dop251/goja"
	"reflect"

	"github.com/smartbch/pureauth/egvm-script/types"
)

type EGVMContext struct {
	config         string
	inputBufLists  [][]byte
	state          []byte
	outputBufLists [][]byte
}

var EGVMCtx *EGVMContext

func GetEGVMContext(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(EGVMCtx)
}

func (e *EGVMContext) Set(job *types.LambdaJob) {
	e.config = job.Config
	e.inputBufLists = job.Inputs
	e.state = job.State
}

func (e *EGVMContext) Reset() {
	e.config = ""
	e.inputBufLists = nil
	e.outputBufLists = nil
	e.state = nil
}

func (e *EGVMContext) CollectResult() *types.LambdaResult {
	return &types.LambdaResult{
		Outputs: e.outputBufLists,
		State:   e.state,
	}
}

func (e *EGVMContext) GetConfig(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(e.config)
}

func (e *EGVMContext) SetConfig(cfg string) {
	e.config = cfg
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
