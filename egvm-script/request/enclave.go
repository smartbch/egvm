package request

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dop251/goja"
	"github.com/edgelesssys/ego/eclient"
	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/pureauth/egvm-script/utils"
)

type attestationResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Result  string `json:"result"`
}

// parameters: serverURL string, signerID string, uniqueID string
// return: isSuccess bool, reason string
func AttestEnclaveServer(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 3 {
		panic(utils.IncorrectArgumentCount)
	}

	serverURL, ok := f.Arguments[0].Export().(string)
	if !ok {
		panic(goja.NewSymbol("The first argument must be server URL"))
	}

	signerID, ok := f.Arguments[1].Export().(string)
	if !ok {
		panic(goja.NewSymbol("The second argument must be signer ID"))
	}

	uniqueID, ok := f.Arguments[2].Export().(string)
	if !ok {
		panic(goja.NewSymbol("The third argument must be unique ID"))
	}

	// 1. get server public key
	pubKeyResp := HttpsRequest(http.MethodGet, fmt.Sprintf("%v/pubkey", serverURL), "", "Content-Type:application/json")
	if pubKeyResp.StatusCode != http.StatusOK {
		panic(goja.NewSymbol(fmt.Sprintf("Error when get server pubkey: %v, body: %v", pubKeyResp.StatusCode, pubKeyResp.Body)))
	}

	var pubKeyResult attestationResult
	err := json.Unmarshal([]byte(pubKeyResp.Body), &pubKeyResult)
	if err != nil {
		panic(goja.NewSymbol("Failed to unmarshal pubKey response: " + err.Error()))
	}

	// 2. get server enclave report
	reportResp := HttpsRequest(http.MethodGet, fmt.Sprintf("%v/pubkey_report", serverURL), "", "Content-Type:application/json")
	if reportResp.StatusCode != http.StatusOK {
		panic(goja.NewSymbol(fmt.Sprintf("Error when get server pubkey: %v, body: %v", reportResp.StatusCode, reportResp.Body)))
	}

	var reportResult attestationResult
	err = json.Unmarshal([]byte(reportResp.Body), &reportResult)
	if err != nil {
		panic(goja.NewSymbol("Failed to unmarshal report response: " + err.Error()))
	}

	// 3. verify
	var result [2]any

	pubKeyBz := gethcmn.FromHex(pubKeyResult.Result)
	reportBz := gethcmn.FromHex(reportResult.Result)
	signerIDBz := gethcmn.FromHex(signerID)
	uniqueIDBz := gethcmn.FromHex(uniqueID)
	err = verifyEnclaveReportBz(pubKeyBz, reportBz, signerIDBz, uniqueIDBz)
	if err != nil {
		result = [2]any{false, err.Error()}
		return vm.ToValue(result)
	}

	result = [2]any{true, ""}
	return vm.ToValue(result)
}

func verifyEnclaveReportBz(pubKey, reportBz, signerIDBz, uniqueIDBz []byte) error {
	report, err := eclient.VerifyRemoteReport(reportBz)
	if err != nil {
		return err
	}

	if !bytes.Equal(report.SignerID, signerIDBz) {
		return fmt.Errorf("signer-id not match! expected: %x, got: %x", signerIDBz, report.SignerID)
	}
	if !bytes.Equal(report.UniqueID, uniqueIDBz) {
		return fmt.Errorf("unique-id not match! expected: %x, got: %x", uniqueIDBz, report.UniqueID)
	}

	hash := sha256.Sum256(pubKey)
	if !bytes.Equal(report.Data[:len(hash)], hash[:]) {
		return errors.New("report data does not match the pubKey's hash")
	}
	if report.SecurityVersion < 2 {
		return errors.New("invalid security version")
	}
	if binary.LittleEndian.Uint16(report.ProductID) != 0x001 {
		return errors.New("invalid product ID")
	}
	if report.Debug {
		return errors.New("should not open debug mode")
	}

	return nil
}
