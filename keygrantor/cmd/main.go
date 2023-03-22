package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	ecies "github.com/ecies/go/v2"
	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/enclave"
	"github.com/tyler-smith/go-bip32"

	"github.com/smartbch/pureauth/keygrantor"
)

var (
	ExtPrivKey *bip32.Key
	ExtPubKey  *bip32.Key
	PrivKey    *ecies.PrivateKey
	SelfReport attestation.Report

	KeyFile = "/data/key.txt"
)

func main() {
	SelfReport = keygrantor.GetSelfReportAndCheck()
	ExtPrivKey = keygrantor.GetRandomExtPrivKey()
	ExtPubKey = ExtPrivKey.PublicKey()
	newKey := keygrantor.NewKeyFromRootKey(ExtPrivKey)
	PrivKey = keygrantor.Bip32KeyToEciesKey(newKey)
	getKey()
	listenAddrP := flag.String("listen", "0.0.0.0:8082", "listen address")
	listenAddr := *listenAddrP
	go createAndStartHttpServer(listenAddr)
	select {}
}

func getKey() {
	keySrc := flag.String("keysrc", "", "the server from which we can sync xprv key")
	if keySrc == nil || len(*keySrc) == 0 {
		keygrantor.SealKeyToFile(KeyFile, ExtPrivKey)
		return
	}
	reportBz, err := enclave.GetRemoteReport(PrivKey.PublicKey.Bytes(true))
	if err != nil {
		fmt.Println("failed to get report attestation report")
		panic(err)
	}
	url := *keySrc+"/getkey?report="+hex.EncodeToString(reportBz)
	encryptedKeyBz := httpGet(url)
	keyBz, err := ecies.Decrypt(PrivKey, encryptedKeyBz)
	if err != nil {
		fmt.Println("failed to decrypt message from server")
		panic(err)
	}
	ExtPrivKey, err = bip32.Deserialize(keyBz)
	if err != nil {
		fmt.Println("failed to deserialize the key from server")
		panic(err)
	}
	ExtPubKey = ExtPrivKey.PublicKey()
	ExtPrivKey = keygrantor.GetRandomExtPrivKey()
	keygrantor.SealKeyToFile(KeyFile, ExtPrivKey)
}

func httpGet(url string) []byte {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("failed to get key, http status: %d", resp.Status))
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return body
}

func createAndStartHttpServer(listenAddr string) {
	http.HandleFunc("/xpub", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(ExtPubKey.B58Serialize()))
	})

	http.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		hash := sha256.Sum256([]byte(ExtPubKey.B58Serialize()))
		report, err := enclave.GetRemoteReport(hash[:])
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte(hex.EncodeToString(report)))
	})

	http.HandleFunc("/getkey", func(w http.ResponseWriter, r *http.Request) {
		reportHex := r.URL.Query()["report"]
		if len(reportHex) == 0 || len(reportHex[0]) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("miss report paramater"))
			return
		}
		reportBz, err := hex.DecodeString(reportHex[0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("report decode error"))
			return
		}
		report, err := keygrantor.VerifyPeerReport(reportBz, SelfReport)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("report check failed"))
			return
		}
		peerPubKey, err := ecies.NewPublicKeyFromBytes(report.Data) // requestor embeds its pubkey here
		var hash [32]byte
		copy(hash[:], report.UniqueID)
		derivedKey := keygrantor.DeriveKey(ExtPrivKey, hash)
		bz, err := ecies.Encrypt(peerPubKey, []byte(derivedKey.B58Serialize()))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to encrypted xprv"))
			return
		}
		w.Write([]byte(hex.EncodeToString(bz)))
	})

	server := http.Server{Addr: listenAddr, ReadTimeout: 3 * time.Second, WriteTimeout: 5 * time.Second}
	fmt.Println("listening ...")
	log.Fatal(server.ListenAndServe())
}
