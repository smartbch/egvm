package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	ecies "github.com/ecies/go/v2"
	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/enclave"
	"github.com/tyler-smith/go-bip32"

	"github.com/smartbch/egvm/keygrantor"
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
	hash := sha256.Sum256(data)
	reportBz, err := enclave.GetRemoteReport(hash[:])
	if err != nil {
		fmt.Println("failed to get report attestation report")
		panic(err)
	}
	token, err := enclave.CreateAzureAttestationToken(data, keygrantor.AttestationProviderURL)
	if err != nil {
		panic(err)
	}
	url := fmt.Sprintf("%s/xprv?pubkey=%s", *keySrc, hex.EncodeToString(data))
	params := keygrantor.GetKeyParams{
		Report: hex.EncodeToString(reportBz),
		JWT:    token,
	}
	jsonReq, err := json.Marshal(params)
	if err != nil {
		panic(err)
	}
	encryptedKey, err := keygrantor.HttpPost(url, jsonReq)
	if err != nil {
		panic(err)
	}
	resBz, err := hex.DecodeString(string(encryptedKey))
	if err != nil {
		panic(err)
	}
	keyBz, err := ecies.Decrypt(PrivKey, resBz)
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
		pubKey, pubkeyBz := handleRequesterPubkey(w, r)
		if pubKey == nil {
			return
		}
		report := handleGetKeyParam(w, r, pubkeyBz)
		if report == nil {
			return
		}
		err := keygrantor.VerifyPeerReport(report, &SelfReport)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("report check failed: " + err.Error()))
			return
		}
		derivedKeyBz, _ := ExtPrivKey.Serialize()
		bz, err := ecies.Encrypt(pubKey, derivedKeyBz)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("failed to encrypted xprv"))
			return
		}
		w.Write([]byte(hex.EncodeToString(bz)))
	})

	// For requestors to get derived key
	http.HandleFunc("/getkey", func(w http.ResponseWriter, r *http.Request) {
		pubKey, pubkeyBz := handleRequesterPubkey(w, r)
		if pubKey == nil {
			return
		}
		report := handleGetKeyParam(w, r, pubkeyBz)
		if report == nil {
			return
		}
		handleKeyDerive(w, report, pubKey)
	})

	server := http.Server{Addr: listenAddr, ReadTimeout: 3 * time.Second, WriteTimeout: 5 * time.Second}
	fmt.Println("listening ...")
	log.Fatal(server.ListenAndServe())
}

func handleRequesterPubkey(w http.ResponseWriter, r *http.Request) (*ecies.PublicKey, []byte) {
	pubkeys := r.URL.Query()["pubkey"]
	if len(pubkeys) == 0 || len(pubkeys[0]) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("miss jwt token parameter"))
		return nil, nil
	}
	requesterPubkeyBz, err := hex.DecodeString(pubkeys[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("pubkey hex string decode error"))
		return nil, nil
	}
	requesterPubKey, err := ecies.NewPublicKeyFromBytes(requesterPubkeyBz) // requester embeds its pubkey here
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("pubkey decode err: " + err.Error()))
		return nil, nil
	}
	return requesterPubKey, requesterPubkeyBz
}

func handleGetKeyParam(w http.ResponseWriter, r *http.Request, pubkeyBz []byte) *attestation.Report {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to read request body"))
		return nil
	}
	var params keygrantor.GetKeyParams
	err = json.Unmarshal(body, &params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed to unmarshal request body"))
		return nil
	}
	reportBz, err := hex.DecodeString(params.Report)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("report decode error"))
		return nil
	}
	report, err := enclave.VerifyRemoteReport(reportBz)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("report check failed: " + err.Error()))
		return nil
	}
	if len(report.Data) != 64 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("report data must 64bytes long"))
		return nil
	}
	pubkeyHash := sha256.Sum256(pubkeyBz)
	if !bytes.Equal(pubkeyHash[:], report.Data[:32]) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("pubkey not match the pubkey hash"))
		return nil
	}
	fmt.Printf("report pubkey: %s\n", hex.EncodeToString(report.Data[:33]))
	err = keygrantor.VerifyJWT(params.JWT, report)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("jwt token verify failed: " + err.Error()))
		return nil
	}
	return &report
}

func handleKeyDerive(w http.ResponseWriter, report *attestation.Report, requesterPubKey *ecies.PublicKey) {
	// concat uniqueid and client specific data, then hash it for more flexible key deriving
	derivedKey := keygrantor.DeriveKey(ExtPrivKey, sha256.Sum256(append(report.UniqueID, report.Data[32:]...)))
	derivedKeyBz, _ := derivedKey.Serialize()
	bz, err := ecies.Encrypt(requesterPubKey, derivedKeyBz)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to encrypted derivedKey"))
		return
	}
	w.Write([]byte(hex.EncodeToString(bz)))
}
