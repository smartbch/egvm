package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/dop251/goja"
	"github.com/smartbch/pureauth/keygrantor"
	"github.com/tinylib/msgp/msgp"
	"github.com/tyler-smith/go-bip32"

	"github.com/smartbch/pureauth/egvm-script/types"
)

var privKey *bip32.Key
var maxMemSize uint64 = 1024 * 1024 * 1024   // 1G
var defaultRunTimeLimitInLoopMode int64 = 30 // 30s

func main() {
	var timeLimitInLoopMode int64
	var singleMode bool
	var perpetualMode bool
	var keygrantorUrl string
	flag.Int64Var(&timeLimitInLoopMode, "t", defaultRunTimeLimitInLoopMode, "enable loop mode: specific run time limit in second")
	flag.BoolVar(&singleMode, "s", false, "enable single mode: accept one input and return one output, then process exit")
	flag.BoolVar(&perpetualMode, "p", false, "enable perpetual mode: not clear the state after script run, accept continuous input")
	flag.StringVar(&keygrantorUrl, "k", "127.0.0.1:8084", "keygrantor url")
	flag.Parse()

	setRlimit(maxMemSize)

	var err error
	privKey, err = keygrantor.GetKeyFromKeyGrantor(keygrantorUrl, [32]byte{})
	if err != nil {
		//panic(err) // comment for core logic test
	}
	if perpetualMode {
		executeLambdaJob(false, true, 0)
	} else if singleMode {
		executeLambdaJob(true, false, 0)
	} else { // loop mode
		executeLambdaJob(false, false, timeLimitInLoopMode)
	}
}

func executeLambdaJob(isSingleMode bool, isPerpetualMode bool, timeLimit int64) {
	for {
		var job types.LambdaJob
		err := job.DecodeMsg(msgp.NewReader(os.Stdin))
		if err != nil {
			panic(err) //todo: log it
		}
		// todo: get output and covert to types.LambdaResult
		_, err = run(job.Script, timeLimit)
		if err != nil {
			handleError(err)
			continue
		}
		var res types.LambdaResult
		bz, _ := res.MarshalMsg(nil)
		_, err = os.Stdout.Write(bz)
		if err != nil {
			panic(err)
		}
		if isSingleMode {
			return
		}
	}
}

func run(script string, timeLimit int64) (goja.Value, error) {
	vm := goja.New()
	RegisterFunctions(vm)
	if timeLimit != 0 {
		var closeChan = make(chan bool)
		defer close(closeChan)
		go func() {
			select {
			case <-time.After(time.Duration(timeLimit) * time.Second):
				vm.Interrupt(errors.New("execution time exceed"))
			case <-closeChan:
				vm.ClearInterrupt()
			}
		}()
	}
	result, err := vm.RunString(script)
	return result, err
}

func handleError(err error) {
	var oErr *goja.Exception
	if errors.As(err, &oErr) {
		fmt.Fprint(os.Stderr, oErr.String())
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
}

func setRlimit(maxMemSize uint64) {
	if runtime.GOOS == "darwin" {
		return
	}
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_AS, &rLimit)
	if err != nil {
		panic(err)
	}
	limit := syscall.Rlimit{Cur: maxMemSize, Max: rLimit.Max}
	err = syscall.Setrlimit(syscall.RLIMIT_AS, &limit)
	if err != nil {
		panic(err)
	}
}
