package extension

import (
	"testing"

	"github.com/dop251/goja"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const (
	AesGcmScriptTemplate = `
		const secret = new Uint8Array([1, 2, 3, 4, 8, 7, 6, 5, 3, 4, 5, 6, 91, 33, 255, 32]).buffer
		const nonce = new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16]).buffer
		const message = new Uint8Array([8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8]).buffer
		const encryptedMsg = AesGcmEncrypt(secret, nonce, message)
		const [decryptedMsg, ok] = AesGcmDecrypt(secret, nonce, encryptedMsg)
	`

	PrivateKeyAndPublicKeyBasicScriptTemplate = `
		const privateKeyHex = 'c9cb992b13141bb3326d028020030f33b92ea9a64b6530291e7876938bd31479'
		const privateKeyBuf = HexToBuf(privateKeyHex)
		const privateKey = BufToPrivateKey(privateKeyBuf)
		
		const privateKeyToBuf = privateKey.Serialize()
		const privateKeyToHex = privateKey.Hex()

		const publicKey = privateKey.GetPublicKey()
		const publicKeyToCompressedBz = publicKey.SerializeCompressed()
		const publicKeyToUncompressedBz = publicKey.SerializeUncompressed()
		
	`
)

func setupGojaVmForCrypto() *goja.Runtime {
	vm := goja.New()
	vm.Set("HexToBuf", HexToBuf)

	vm.Set("AesGcmDecrypt", AesGcmDecrypt)
	vm.Set("AesGcmEncrypt", AesGcmEncrypt)

	vm.Set("BufToPrivateKey", BufToPrivateKey)
	vm.Set("BufToPublicKey", BufToPublicKey)
	return vm
}

func TestAesGcm(t *testing.T) {
	vm := setupGojaVmForCrypto()
	_, err := vm.RunString(AesGcmScriptTemplate)
	require.NoError(t, err)

	encryptedMsg := vm.Get("encryptedMsg").Export().(goja.ArrayBuffer)
	encryptedMsgHex := gethcmn.Bytes2Hex(encryptedMsg.Bytes())
	decryptedMsg := vm.Get("decryptedMsg").Export().(goja.ArrayBuffer)
	decryptedMsgHex := gethcmn.Bytes2Hex(decryptedMsg.Bytes())
	ok := vm.Get("ok").Export().(bool)
	require.True(t, ok)
	require.EqualValues(t, "fe5695ffe3ce8fb998bb9e35f529507b2c990e7de5c5d99e32d1f546130c6ef4", encryptedMsgHex)
	require.EqualValues(t, "08080808080808080808080808080808", decryptedMsgHex)

}

func TestPrivateKeyAndPublicKeyBasic(t *testing.T) {
	vm := setupGojaVmForCrypto()
	_, err := vm.RunString(PrivateKeyAndPublicKeyBasicScriptTemplate)
	require.NoError(t, err)

	privateKeyToBuf := vm.Get("privateKeyToBuf").Export().(goja.ArrayBuffer)
	privateKeyToBufHex := gethcmn.Bytes2Hex(privateKeyToBuf.Bytes())
	privateKeyToHex := vm.Get("privateKeyToHex").Export().(string)
	require.EqualValues(t, "c9cb992b13141bb3326d028020030f33b92ea9a64b6530291e7876938bd31479", privateKeyToBufHex)
	require.EqualValues(t, "c9cb992b13141bb3326d028020030f33b92ea9a64b6530291e7876938bd31479", privateKeyToHex)

	publicKeyToCompressedBz := vm.Get("publicKeyToCompressedBz").Export().(goja.ArrayBuffer)
	publicKeyToCompressedBzHex := gethcmn.Bytes2Hex(publicKeyToCompressedBz.Bytes())
	publicKeyToUncompressedBz := vm.Get("publicKeyToUncompressedBz").Export().(goja.ArrayBuffer)
	publicKeyToUncompressedBzHex := gethcmn.Bytes2Hex(publicKeyToUncompressedBz.Bytes())
	require.EqualValues(t, "02dde6c067f5e1a641dedab654cbbd9c3b4c6f8adbf2aeb17c6500319d2c08f08e", publicKeyToCompressedBzHex)
	require.EqualValues(t, "04dde6c067f5e1a641dedab654cbbd9c3b4c6f8adbf2aeb17c6500319d2c08f08ec14990f4beca82491ca7134e7c637a35073df4e4ef8ee4251f5b4570f222777a",
		publicKeyToUncompressedBzHex)
}
