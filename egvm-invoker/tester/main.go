package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smartbch/pureauth/egvm-script/types"
)

var invokerUrl = "http://127.0.0.1:8001/execute"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var job types.LambdaJob
	job.Script = "1 + 1"
	body, _ := job.MarshalMsg(nil)
	req := sendGzipRequest(ctx, http.MethodPost, invokerUrl, body)
	defer req.Body.Close()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	bz, err := io.ReadAll(resp.Body)
	var res types.LambdaResult
	_, err = res.UnmarshalMsg(bz)
	if err != nil {
		panic(err)
	}
	out, _ := json.Marshal(res)
	fmt.Println(string(out))
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
