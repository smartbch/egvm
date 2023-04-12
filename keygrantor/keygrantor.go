package keygrantor

import "C"
import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"time"

	ecies "github.com/ecies/go/v2"
	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/attestation/tcbstatus"
	"github.com/edgelesssys/ego/ecrypto"
	"github.com/edgelesssys/ego/enclave"
	"github.com/tyler-smith/go-bip32"
)

//// #include "util.h"
//import "C"

var (
	ExtPrivKey *bip32.Key
	ExtPubKey  *bip32.Key
	PrivKey    *ecies.PrivateKey

	ErrInDebugMode       = errors.New("Cannot work in debug mode")
	ErrTCBStatus         = errors.New("TCB is not up-to-date")
	ErrUniqueIDMismatch  = errors.New("UniqueID Mismatch")
	ErrSignerIDMismatch  = errors.New("SignerID Mismatch")
	ErrProductIDMismatch = errors.New("ProductID Mismatch")
)

type GetKeyParams struct {
	Report string `json:"Report"`
	JWT    string `json:"JWT"`
}

func generateRandom64Bytes() []byte {
	var out []byte
	//var x C.uint16_t
	//var retry C.int = 1
	for i := 0; i < 64; i++ {
		//C.rdrand16(&x, retry)
		//out = append(out, byte(x))
	}
	return out
}

func generateRandom32Bytes() []byte {
	var out []byte
	//var x C.uint16_t
	//var retry C.int = 1
	for i := 0; i < 32; i++ {
		//C.rdrand16(&x, retry)
		//out = append(out, byte(x))
	}
	return out
}

func GetRandomExtPrivKey() *bip32.Key {
	seed := generateRandom64Bytes()
	key, err := bip32.NewMasterKey(seed)
	if err != nil {
		panic(err)
	}
	return key
}

func Bip32KeyToEciesKey(key *bip32.Key) *ecies.PrivateKey {
	return ecies.NewPrivateKeyFromBytes(key.Key)
}

func DeriveKey(key *bip32.Key, hash [32]byte) *bip32.Key {
	twoExp31 := big.NewInt(1 << 31)
	n := big.NewInt(0).SetBytes(hash[:])
	lastAdd := uint32(0)
	lastAddUnit := uint32(1 << 8)
	for i := 0; i < 9; i++ {
		remainder := big.NewInt(0)
		n.DivMod(n, twoExp31, remainder)
		for m := uint32(remainder.Uint64()); true; m++ {
			//fmt.Printf("i %d m %08x\n", i, m)
			var err error
			if i == 8 {
				key, err = key.NewChildKey(m)
			} else { //last round
				key, err = key.NewChildKey(m + lastAdd)
			}
			if err == nil {
				break
			} else { // very unlikely
				lastAdd += lastAddUnit
			}
		}
		lastAddUnit <<= 3
	}
	return key
}

func NewKeyFromRootKey(rootKey *bip32.Key) *bip32.Key {
	child, err := rootKey.NewChildKey(0x80000000 + 44) // BIP44
	if err != nil {
		panic(err)
	}
	child, err = child.NewChildKey(0x80000000) //Bitcoin
	if err != nil {
		panic(err)
	}
	child, err = child.NewChildKey(0) //account=0
	if err != nil {
		panic(err)
	}
	child, err = child.NewChildKey(0) //chain=0
	if err != nil {
		panic(err)
	}
	child, err = child.NewChildKey(0) //address=0
	if err != nil {
		panic(err)
	}
	return child
}

func SealKeyToFile(fname string, extPrivKey *bip32.Key) {
	bz, err := extPrivKey.Serialize()
	if err != nil {
		panic(err)
	}
	out, err := ecrypto.SealWithUniqueKey(bz, nil)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(fname, out, 0600)
	if err != nil {
		panic(err)
	}
}

func RecoverKeysFromFile(fname string) (extPrivKey *bip32.Key, extPubKey *bip32.Key, privKey *ecies.PrivateKey, err error) {
	fileData, err := os.ReadFile(fname)
	if err != nil {
		fmt.Printf("read file failed, %s\n", err.Error())
		if os.IsNotExist(err) {
			return nil, nil, nil, err
		}
		panic(err)
	}
	rawData, err := ecrypto.Unseal(fileData, nil)
	if err != nil {
		fmt.Printf("unseal file data failed, %s\n", err.Error())
		panic(err)
	}
	extPrivKey, err = bip32.Deserialize(rawData)
	if err != nil {
		fmt.Printf("deserialize xprv failed, %s\n", err.Error())
		panic(err)
	}
	extPubKey = extPrivKey.PublicKey()
	privKey = Bip32KeyToEciesKey(NewKeyFromRootKey(extPrivKey))
	return
}

func GetSelfReportAndCheck() attestation.Report {
	report, err := enclave.GetSelfReport()
	if err != nil {
		panic(err)
	}
	if report.Debug {
		panic(ErrInDebugMode)
	}
	r, err := enclave.GetRemoteReport([]byte{0x01})
	if err != nil {
		panic(err)
	}
	ar, err := enclave.VerifyRemoteReport(r)
	if err != nil {
		panic(err)
	}
	if ar.TCBStatus != tcbstatus.UpToDate {
		panic(ErrTCBStatus)
	}
	return report
}

func VerifyPeerReport(report *attestation.Report, selfReport *attestation.Report) error {
	if report.Debug {
		return ErrInDebugMode
	}
	if report.TCBStatus != tcbstatus.UpToDate {
		return ErrTCBStatus
	}
	if !bytes.Equal(selfReport.UniqueID, report.UniqueID) {
		return ErrUniqueIDMismatch
	}
	if !bytes.Equal(selfReport.SignerID, report.SignerID) {
		return ErrSignerIDMismatch
	}
	if !bytes.Equal(selfReport.ProductID, report.ProductID) {
		return ErrProductIDMismatch
	}
	return nil
}

func HttpPost(url string, jsonReq []byte) ([]byte, error) {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(jsonReq))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get key, http status:%s, content:%s", resp.Status, string(body))
	}
	return body, nil
}

func GetKeyFromKeyGrantor(keyGrantorUrl string, clientData [32]byte) (*bip32.Key, error) {
	privKey := ecies.NewPrivateKeyFromBytes(generateRandom32Bytes())
	pubkey := privKey.PublicKey.Bytes(true)
	pubkeyHash := sha256.Sum256(pubkey)
	report, err := enclave.GetRemoteReport(append(pubkeyHash[:], clientData[:]...))
	if err != nil {
		return nil, fmt.Errorf("failed to get remote report: %w", err)
	}
	token, err := enclave.CreateAzureAttestationToken(pubkey, AttestationProviderURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create attestation report: %w", err)
	}
	url := fmt.Sprintf("%s/getkey?pubkey=%s", keyGrantorUrl, hex.EncodeToString(pubkey))
	params := GetKeyParams{
		Report: hex.EncodeToString(report),
		JWT:    token,
	}
	jsonReq, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	res, err := HttpPost(url, jsonReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}
	if res == nil {
		return nil, fmt.Errorf("failed to get key: no resust data")
	}
	resBz, err := hex.DecodeString(string(res))
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}
	keyBz, err := ecies.Decrypt(privKey, resBz)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt message from server: %w", err)
	}
	outKey, err := bip32.Deserialize(keyBz)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize the key from server: %w", err)
	}
	return outKey, nil
}

const AttestationProviderURL = "https://shareduks.uks.attest.azure.net"

func VerifyJWT(token string, report attestation.Report) error {
	tokenReport, err := attestation.VerifyAzureAttestationToken(token, AttestationProviderURL)
	if err != nil {
		return err
	}
	return checkJWTAgainstReport(tokenReport, report)
}

func checkJWTAgainstReport(token attestation.Report, report attestation.Report) error {
	if !bytes.Equal(token.UniqueID, report.UniqueID) {
		return ErrUniqueIDMismatch
	}
	if !bytes.Equal(token.SignerID, report.SignerID) {
		return ErrSignerIDMismatch
	}
	if !bytes.Equal(token.ProductID, report.ProductID) {
		return ErrProductIDMismatch
	}
	return nil
}
