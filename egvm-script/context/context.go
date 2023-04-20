package context

import "github.com/smartbch/pureauth/egvm-script/types"

type EGVMContext struct {
	Config         string
	InputBufLists  [][]byte
	State          []byte // using New OrderedMapReader(EGVMContext.State).read() to get maps in js
	OutputBufLists [][]byte
}

var EGVMCtx *EGVMContext

func (e *EGVMContext) Set(job *types.LambdaJob) {
	e.Config = job.Config
	e.InputBufLists = job.Inputs
	e.State = job.State
}

func (e *EGVMContext) Reset() {
	e.Config = ""
	e.InputBufLists = nil
	e.OutputBufLists = nil
	e.State = nil
}

func (e *EGVMContext) CollectResult() *types.LambdaResult {
	return &types.LambdaResult{
		Outputs: e.OutputBufLists,
		State:   e.State,
	}
}
