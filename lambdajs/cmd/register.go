package main

import (
	"github.com/dop251/goja"

	"github.com/smartbch/pureauth/lambdajs/extension"
	"github.com/smartbch/pureauth/lambdajs/request"
	"github.com/smartbch/pureauth/lambdajs/types"
)

func RegisterFunctions(vm *goja.Runtime) {
	// types
	vm.Set("HexToU256", types.HexToU256)
	vm.Set("BufToU256", types.BufToU256)
	vm.Set("HexToS256", types.HexToS256)
	vm.Set("BufToS256", types.BufToS256)
	vm.Set("U256", types.U256)
	vm.Set("S256", types.S256)

	// extension functions
	vm.Set("AesGcmDecrypt", extension.AesGcmDecrypt)
	vm.Set("AesGcmEncrypt", extension.AesGcmEncrypt)
	vm.Set("BufToPrivateKey", extension.BufToPrivateKey)
	vm.Set("BufToPublicKey", extension.BufToPublicKey)
	vm.Set("Keccak256", extension.Keccak256)
	vm.Set("Sha256", extension.Sha256)
	vm.Set("Ripemd160", extension.Ripemd160)
	vm.Set("BufConcat", extension.BufConcat)
	vm.Set("HexToBuf", extension.HexToBuf)
	vm.Set("B64ToBuf", extension.B64ToBuf)
	vm.Set("BufToB64", extension.BufToB64)
	vm.Set("BufToHex", extension.BufToHex)
	vm.Set("BufEqual", extension.BufEqual)
	vm.Set("BufCompare", extension.BufCompare)
	vm.Set("VerifySignature", extension.VerifySignature)
	vm.Set("Ecrecover", extension.Ecrecover)
	vm.Set("ZstdCompress", extension.ZstdCompress)
	vm.Set("ZstdDecompress", extension.ZstdDecompress)
	vm.Set("VerifyMerkleProofSha256", extension.VerifyMerkleProofSha256)
	vm.Set("VerifyMerkleProofKeccak256", extension.VerifyMerkleProofKeccak256)
	vm.Set("SignTxAndSerialize", extension.SignTxAndSerialize)
	vm.Set("ParseTxInHex", extension.ParseTxInHex)

	// http request
	vm.Set("HttpRequest", request.HttpRequest)
	vm.Set("HttpsRequest", request.HttpsRequest)

}
