package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"github.com/tinylib/msgp/msgp"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smartbch/pureauth/egvm-script/types"
)

var invokerUrl = "http://127.0.0.1:8001/execute"

func main() {
	var scriptFile string
	var certFiles string
	var config string
	var inputString string
	var stateString string
	flag.StringVar(&scriptFile, "w", "", "script file")
	flag.StringVar(&certFiles, "f", "", "cert files separated with comma")
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
	for _, certFile := range strings.Split(certFiles, ",") {
		if certFile != "" {
			certB, err := os.ReadFile(certFile)
			if err != nil {
				panic(err)
			}
			job.Certs = append(job.Certs, string(certB))
		}
	}
	job.Config = config
	body, _ := job.MarshalMsg(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req := sendGzipRequest(ctx, http.MethodPost, invokerUrl, body)
	defer req.Body.Close()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		panic(err)
	}
	//bz, _ := ioutil.ReadAll(gr)
	//fmt.Println(string(bz))
	var res types.LambdaResult
	err = res.DecodeMsg(msgp.NewReader(gr))
	if err != nil {
		panic(err)
	}
	for _, out := range res.Outputs {
		fmt.Println(string(out))
	}
}

func sendGzipRequest(ctx context.Context, method, url string, body any) *http.Request {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(body.([]byte))
	if err != nil {
		panic(err)
	}
	gz.Close()
	r, err := http.NewRequestWithContext(ctx, method, url, &b)
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Encoding", "gzip")
	return r
}
