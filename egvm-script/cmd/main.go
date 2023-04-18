package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dop251/goja"
	"github.com/smartbch/pureauth/keygrantor"
	"github.com/tinylib/msgp/msgp"
	"github.com/tyler-smith/go-bip32"

	"github.com/smartbch/pureauth/egvm-script/types"
)

var privKey *bip32.Key

func main() {
	var runMode string
	var keygrantorUrl string
	flag.StringVar(&runMode, "m", "once", "run mode: once (default), keepalive")
	flag.StringVar(&keygrantorUrl, "k", "127.0.0.1:8084", "keygrantor url")
	flag.Parse()
	var err error
	privKey, err = keygrantor.GetKeyFromKeyGrantor(keygrantorUrl, [32]byte{})
	if err != nil {
		//panic(err) // comment for core logic test
	}
	if runMode == "keepalive" {
		runForever()
	} else {
		src, err := readSource(flag.Arg(0))
		if err != nil {
			handleError(err)
			os.Exit(64)
		}
		out, err := run(string(src))
		if err != nil {
			handleError(err)
			os.Exit(64)
		}
		fmt.Println(out)
	}
}

func runForever() {
	for {
		var job types.LambdaJob
		err := job.DecodeMsg(msgp.NewReader(os.Stdin))
		if err != nil {
			panic(err) //todo: log it
		}
		// todo: get output and covert to types.LambdaResult
		_, err = run(job.Script)
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
		return
	}
}

func run(script string) (goja.Value, error) {
	vm := goja.New()
	RegisterFunctions(vm)
	result, err := vm.RunString(script)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func handleError(err error) {
	var oErr *goja.Exception
	if errors.As(err, &oErr) {
		fmt.Fprint(os.Stderr, oErr.String())
	} else {
		fmt.Fprintln(os.Stderr, err)
	}
}

func readSource(filename string) ([]byte, error) {
	if filename == "" || filename == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(filename) //nolint: gosec
}
