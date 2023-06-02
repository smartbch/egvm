package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/smartbch/egvm/egvm-invoker/executor"
	"github.com/smartbch/egvm/egvm-script/types"
)

func main() {
	var listenAddr string
	flag.StringVar(&listenAddr, "l", "127.0.0.1:8001", "listen address")
	flag.Parse()
	m := executor.NewSandboxManager(nil)
	addHttpHandler(m)
	server := http.Server{Addr: listenAddr, ReadTimeout: 3 * time.Second, WriteTimeout: 5 * time.Second}
	fmt.Println("listening ...")
	log.Fatal(server.ListenAndServe())
}

func addHttpHandler(m *executor.SandboxManager) {
	http.HandleFunc("/execute", func(w http.ResponseWriter, r *http.Request) {
		uncompressedBody, err := gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			gzipWrite(w, []byte("failed to uncompress request body"))
			return
		}
		body, err := ioutil.ReadAll(uncompressedBody)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			gzipWrite(w, []byte("failed to read request body"))
			return
		}
		var job types.LambdaJob
		_, err = job.UnmarshalMsg(body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			gzipWrite(w, []byte("failed to unmarshal request body"))
			return
		}
		result, err := m.ExecuteJob(&job)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			gzipWrite(w, []byte("failed to execute lambda job:"+err.Error()))
			return
		}
		out, err := result.MarshalMsg(nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			gzipWrite(w, []byte("failed to marshal result body"))
			return
		}
		gzipWrite(w, out)
		return
	})
}

func gzipWrite(w http.ResponseWriter, content []byte) {
	gw := gzip.NewWriter(w)
	gw.Write(content)
	gw.Close()
}
