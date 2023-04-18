package main

import (
	"encoding/hex"
	"flag"
	"os"
	"strings"

	"github.com/smartbch/pureauth/egvm-script/types"
)

func main() {
	var scriptFile string
	var certFile string
	var config string
	var inputString string
	var stateString string
	flag.StringVar(&scriptFile, "w", "", "script file")
	flag.StringVar(&certFile, "f", "", "cert file")
	flag.StringVar(&config, "c", "", "config")
	flag.StringVar(&inputString, "i", "", "hex encoded input separated with comma")
	flag.StringVar(&stateString, "s", "", "hex encoded state")
	flag.Parse()
	var job types.LambdaJob
	scriptB, err := os.ReadFile(scriptFile)
	if err != nil {
		panic(err)
	}
	job.Script = string(scriptB)
	//fmt.Println(job.Script)
	if certFile != "" {
		certB, err := os.ReadFile(certFile)
		if err != nil {
			panic(err)
		}
		job.Cert = string(certB)
	}
	job.Config = config
	var inputs [][]byte
	for _, i := range strings.Split(inputString, ",") {
		input, err := hex.DecodeString(i)
		if err != nil {
			panic(err)
		}
		inputs = append(inputs, input)
	}
	job.Inputs = inputs
	state, err := hex.DecodeString(stateString)
	if err != nil {
		panic(err)
	}
	job.State = state
	bz, err := job.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(bz)
}
