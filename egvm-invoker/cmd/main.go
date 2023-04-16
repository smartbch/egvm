package main

import (
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/smartbch/pureauth/egvm-invoker/executor"
	"github.com/smartbch/pureauth/egvm-script/types"
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
			w.Write([]byte("failed to uncompress request body"))
			return
		}
		body, err := ioutil.ReadAll(uncompressedBody)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to read request body"))
			return
		}
		var job types.LambdaJob
		err = json.Unmarshal(body, &job)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("failed to unmarshal request body"))
			return
		}
		result, err := m.ExecuteJob(&job)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to execute lambda job"))
			return
		}
		out, err := json.Marshal(result)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to marshal result body"))
			return
		}
		//todo: gzip the response
		w.Write([]byte(hex.EncodeToString(out)))
		return
	})
}
