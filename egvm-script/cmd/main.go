package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/smartbch/pureauth/keygrantor"
	"io"
	"os"

	"github.com/dop251/goja"
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
		_, err = run(string(src))
		if err != nil {
			handleError(err)
			os.Exit(64)
		}
	}
}

func runForever() {
	for {
		var script []byte
		reader := bufio.NewReader(os.Stdin)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 128*1024*1024)
		for scanner.Scan() {
			script = scanner.Bytes()
			break
		}
		var job types.LambdaJob
		_, err := job.UnmarshalMsg(script)
		if err != nil {
			fmt.Println(err)
		}
		// todo: get output and covert to types.LambdaResult
		_, err = run(job.Script)
		if err != nil {
			handleError(err)
			continue
		}
		var res types.LambdaResult
		bz, _ := res.MarshalMsg(nil)
		fmt.Println(string(bz)) // write result to stdout
	}
}

func run(script string) (goja.Value, error) {
	vm := goja.New()
	RegisterFunctions(vm)
	result, err := vm.RunString(script)
	if err != nil {
		return result, err
	}
	return nil, nil
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
