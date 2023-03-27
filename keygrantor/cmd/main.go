package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
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
	keySrc := flag.String("xprvsrc", "", "the server from which we can sync xprv key")
	listenAddrP := flag.String("listen", "0.0.0.0:8084", "listen address")
	flag.Parse()
	var err error
	ExtPrivKey, ExtPubKey, PrivKey, err = keygrantor.RecoverKeysFromFile(KeyFile)
	if err != nil {
		ExtPrivKey = keygrantor.GetRandomExtPrivKey()
		ExtPubKey = ExtPrivKey.PublicKey()
		PrivKey = keygrantor.Bip32KeyToEciesKey(keygrantor.NewKeyFromRootKey(ExtPrivKey))
		fetchXprv(keySrc)
	}
	SelfReport = keygrantor.GetSelfReportAndCheck()
	listenAddr := *listenAddrP
	go createAndStartHttpServer(listenAddr)
	select {}
}

func fetchXprv(keySrc *string) {
	if keySrc == nil || len(*keySrc) == 0 {
		keygrantor.SealKeyToFile(KeyFile, ExtPrivKey)
		return
	}
	data := PrivKey.PublicKey.Bytes(true)
	reportBz, err := enclave.GetRemoteReport(data)
	if err != nil {
		fmt.Println("failed to get report attestation report")
		panic(err)
	}
	token, err := enclave.CreateAzureAttestationToken(data, keygrantor.AttestationProviderURL)
	if err != nil {
		panic(err)
	}
	url := fmt.Sprintf("%s/xprv?report=%s&token=%s", *keySrc, hex.EncodeToString(reportBz), token)
	encryptedKeyBz := keygrantor.HttpGet(url)
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
	//ExtPrivKey = keygrantor.GetRandomExtPrivKey()
	keygrantor.SealKeyToFile(KeyFile, ExtPrivKey)
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

	// For peer keygrantors to get ExtPrivKey
	http.HandleFunc("/xprv", func(w http.ResponseWriter, r *http.Request) {
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
			w.Write([]byte("report check failed: " + err.Error()))
			return
		}
		if len(report.Data) != 64 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("report data must 64bytes long"))
			return
		}
		peerPubKey, err := ecies.NewPublicKeyFromBytes(report.Data[:33]) // requestor embeds its pubkey here
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("report data must be pubkey: " + err.Error()))
			return
		}
		tokens := r.URL.Query()["token"]
		if len(tokens) == 0 || len(tokens[0]) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("miss jwt token parameter"))
			return
		}
		err = keygrantor.VerifyJWT(tokens[0], report)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("jwt token verify failed: " + err.Error()))
			return
		}
		derivedKeyBz, _ := ExtPrivKey.Serialize()
		bz, err := ecies.Encrypt(peerPubKey, derivedKeyBz)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to encrypted xprv"))
			return
		}
		w.Write([]byte(hex.EncodeToString(bz)))
	})

	// For requestors to get derived key
	http.HandleFunc("/getkey", func(w http.ResponseWriter, r *http.Request) {
		// do verify
		reportHex := r.URL.Query()["report"]
		if len(reportHex) == 0 || len(reportHex[0]) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("miss report parameter"))
			return
		}
		reportBz, err := hex.DecodeString(reportHex[0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("report decode error"))
			return
		}
		report, err := enclave.VerifyRemoteReport(reportBz)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("report check failed: " + err.Error()))
			return
		}
		if len(report.Data) != 64 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("report data must 64bytes long"))
			return
		}
		fmt.Printf("report pubkey: %s\n", hex.EncodeToString(report.Data[:33]))
		requestorPubKey, err := ecies.NewPublicKeyFromBytes(report.Data[:33]) // requestor embeds its pubkey here
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("report data must be pubkey: " + err.Error()))
			return
		}
		tokens := r.URL.Query()["token"]
		if len(tokens) == 0 || len(tokens[0]) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("miss jwt token parameter"))
			return
		}
		err = keygrantor.VerifyJWT(tokens[0], report)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("jwt token verify failed: " + err.Error()))
			return
		}
		// do key derive
		var hash [32]byte
		copy(hash[:], report.UniqueID)
		derivedKey := keygrantor.DeriveKey(ExtPrivKey, hash)
		derivedKeyBz, _ := derivedKey.Serialize()
		bz, err := ecies.Encrypt(requestorPubKey, derivedKeyBz)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to encrypted derivedKey for requestor"))
			return
		}
		w.Write([]byte(hex.EncodeToString(bz)))
	})

	server := http.Server{Addr: listenAddr, ReadTimeout: 3 * time.Second, WriteTimeout: 5 * time.Second}
	fmt.Println("listening ...")
	log.Fatal(server.ListenAndServe())
}
