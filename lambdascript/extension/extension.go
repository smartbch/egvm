package extension

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/dop251/goja"
	ecies "github.com/ecies/go/v2"
	"github.com/ethereum/go-ethereum/crypto"
)

func RegisterFunctions(vm *goja.Runtime) {
	vm.Set("HexToU256", HexToU256)
	vm.Set("BufToU256", BufToU256)
	vm.Set("HexToS256", HexToS256)
	vm.Set("BufToS256", BufToS256)
	vm.Set("U256", U256)
	vm.Set("S256", S256)
	vm.Set("BufToPrivateKey", BufToPrivateKey)
	vm.Set("BufToPublicKey", BufToPublicKey)
	vm.Set("Keccak256", Keccak256)
	vm.Set("Sha256", Sha256)
	vm.Set("BufConcat", BufConcat)
	vm.Set("HexToBuf", HexToBuf)
	vm.Set("B64ToBuf", B64ToBuf)
	vm.Set("BufToB64", BufToB64)
	vm.Set("BufToHex", BufToHex)
	vm.Set("BufEqual", BufEqual)
	vm.Set("BufCompare", BufCompare)
	vm.Set("VerifySignature", VerifySignature)
	vm.Set("Ecrecover", Ecrecover)
}

var (
	IncorrectArgumentCount = goja.NewSymbol("incorrect argument count")
)

func getOneArrayBuffer(f goja.FunctionCall) []byte {
	if len(f.Arguments) != 1 {
		panic(IncorrectArgumentCount)
	}
	a, ok := f.Arguments[0].Export().(goja.ArrayBuffer)
	if !ok {
		panic(goja.NewSymbol("The first argument must be ArrayBuffer"))
	}
	return a.Bytes()
}

func getTwoArrayBuffers(f goja.FunctionCall) ([]byte, []byte) {
	if len(f.Arguments) != 2 {
		panic(IncorrectArgumentCount)
	}
	a, ok := f.Arguments[0].Export().(goja.ArrayBuffer)
	if !ok {
		panic(goja.NewSymbol("The first argument must be ArrayBuffer"))
	}
	b, ok := f.Arguments[1].Export().(goja.ArrayBuffer)
	if !ok {
		panic(goja.NewSymbol("The second argument must be ArrayBuffer"))
	}
	return a.Bytes(), b.Bytes()
}

func getThreeArrayBuffers(f goja.FunctionCall) ([]byte, []byte, []byte) {
	if len(f.Arguments) != 3 {
		panic(IncorrectArgumentCount)
	}
	a, ok := f.Arguments[0].Export().(goja.ArrayBuffer)
	if !ok {
		panic(goja.NewSymbol("The first argument must be ArrayBuffer"))
	}
	b, ok := f.Arguments[1].Export().(goja.ArrayBuffer)
	if !ok {
		panic(goja.NewSymbol("The second argument must be ArrayBuffer"))
	}
	c, ok := f.Arguments[2].Export().(goja.ArrayBuffer)
	if !ok {
		panic(goja.NewSymbol("The second argument must be ArrayBuffer"))
	}
	return a.Bytes(), b.Bytes(), c.Bytes()
}

func aesgcmEncrypt(secret, nonce, msg []byte) ([]byte, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, fmt.Errorf("cannot create new aes block: %w", err)
	}

	aesgcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return nil, fmt.Errorf("cannot create aes gcm: %w", err)
	}

	return aesgcm.Seal(nil, nonce, msg, nil), nil
}

func aesgcmDecrypt(secret, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, fmt.Errorf("cannot create new aes block: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return nil, fmt.Errorf("cannot create gcm cipher: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot decrypt ciphertext: %w", err)
	}

	return plaintext, nil
}

func AesGcmEncrypt(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	secret, nonce, msg := getThreeArrayBuffers(f)
	bz, err := aesgcmEncrypt(secret, nonce, msg)
	if err != nil {
		panic(goja.NewSymbol("error in AesGcmEncrypt: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

func AesGcmDecrypt(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	secret, nonce, ciphertext := getThreeArrayBuffers(f)
	bz, err := aesgcmDecrypt(secret, nonce, ciphertext)
	if err != nil {
		panic(goja.NewSymbol("error in AesGcmDecrypt: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

// ===================================

type PrivateKey struct {
	key *ecies.PrivateKey
}

type PublicKey struct {
	key *ecies.PublicKey
}

func BufToPrivateKey(buf goja.ArrayBuffer) PrivateKey {
	key := ecies.NewPrivateKeyFromBytes(buf.Bytes())
	return PrivateKey{key: key}
}

func BufToPublicKey(buf goja.ArrayBuffer) PublicKey {
	key, _ := ecies.NewPublicKeyFromBytes(buf.Bytes())
	return PublicKey{key: key}
}

func (prv PrivateKey) GetPublicKey() PublicKey {
	return PublicKey{key: prv.key.PublicKey}
}

func (prv PrivateKey) ECDH(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 1 {
		panic(IncorrectArgumentCount)
	}
	pub, ok := f.Arguments[0].Export().(PublicKey)
	if !ok {
		panic(goja.NewSymbol("The first argument must be PublicKey"))
	}
	bz, err := prv.key.ECDH(pub.key)
	if err != nil {
		panic(goja.NewSymbol("error in ECDH: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

func (prv PrivateKey) Encapsulate(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 1 {
		panic(IncorrectArgumentCount)
	}
	pub, ok := f.Arguments[0].Export().(PublicKey)
	if !ok {
		panic(goja.NewSymbol("The first argument must be PublicKey"))
	}
	bz, err := prv.key.Encapsulate(pub.key)
	if err != nil {
		panic(goja.NewSymbol("error in Encapsulate: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

func (prv PrivateKey) Sign(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	msg := getOneArrayBuffer(f)
	ecdsaPrv := &ecdsa.PrivateKey {
		D: prv.key.D,
		PublicKey: ecdsa.PublicKey {
			Curve: prv.key.PublicKey.Curve,
			X: prv.key.PublicKey.X,
			Y: prv.key.PublicKey.Y,
		},
	}
	sig, err := crypto.Sign(msg, ecdsaPrv)
	if err != nil {
		panic(goja.NewSymbol("error in Sign: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(sig))
}

func (prv PrivateKey) Equal(other PrivateKey) bool {
	return prv.key.Equals(other.key)
}

func (prv PrivateKey) Hex() string {
	return prv.key.Hex()
}

func (prv PrivateKey) ToBuf(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(IncorrectArgumentCount)
	}
	return vm.ToValue(vm.NewArrayBuffer(prv.key.Bytes()))
}

func (pub PublicKey) Decapsulate(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 1 {
		panic(IncorrectArgumentCount)
	}
	prv, ok := f.Arguments[0].Export().(PrivateKey)
	if !ok {
		panic(goja.NewSymbol("The first argument must be PrivateKey"))
	}
	bz, err := pub.key.Decapsulate(prv.key)
	if err != nil {
		panic(goja.NewSymbol("error in Decapsulate: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

func (pub PublicKey) Equal(other PublicKey) bool {
	return pub.key.Equals(other.key)
}

func (pub PublicKey) Hex(compressed bool) string {
	return pub.key.Hex(compressed)
}

func (pub PublicKey) ToBuf(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 1 {
		panic(IncorrectArgumentCount)
	}
	compressed, ok := f.Arguments[0].Export().(bool)
	if !ok {
		panic(goja.NewSymbol("The first argument must be boolean"))
	}
	return vm.ToValue(vm.NewArrayBuffer(pub.key.Bytes(compressed)))
}

func (prv PrivateKey) Decrypt(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	buf := getOneArrayBuffer(f)
	bz, err := ecies.Decrypt(prv.key, buf)
	if err != nil {
		panic(goja.NewSymbol("error in Decrypt: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

func (pub PublicKey) Encrypt(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	buf, entropy := getTwoArrayBuffers(f)
	bz, err := eciesEncrypt(pub.key, buf, entropy)
	if err != nil {
		panic(goja.NewSymbol("error in Encrypt: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

func eciesEncrypt(pubkey *ecies.PublicKey, msg, entropy []byte) ([]byte, error) {
	var ct bytes.Buffer

	// Generate ephemeral key
	h := sha256.New()
	h.Write(entropy)
	h.Write(msg)
	ek := ecies.NewPrivateKeyFromBytes(h.Sum(nil))

	ct.Write(ek.PublicKey.Bytes(false))

	// Derive shared secret
	secret, err := ek.Encapsulate(pubkey)
	if err != nil {
		return nil, err
	}

	nonce := h.Sum(entropy)[:16]

	ct.Write(nonce)

	ciphertext, err := aesgcmEncrypt(secret, nonce, msg)
	if err != nil {
		return nil, err
	}

	tag := ciphertext[len(ciphertext)-16:]
	ct.Write(tag)
	ciphertext = ciphertext[:len(ciphertext)-len(tag)]
	ct.Write(ciphertext)

	return ct.Bytes(), nil
}

func (pub PublicKey) ToAddress(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(IncorrectArgumentCount)
	}
	key := ecdsa.PublicKey {
		Curve: pub.key.Curve,
		X: pub.key.X,
		Y: pub.key.Y,
	}
	addr := crypto.PubkeyToAddress(key)
	return vm.ToValue(vm.NewArrayBuffer(addr[:]))
}

// ===============

func Keccak256(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	data := [][]byte{}
	for _, arg := range(f.Arguments) {
		switch v := arg.Export().(type) {
		case goja.ArrayBuffer:
			data = append(data, v.Bytes());
		case Uint256:
			dest := [32]byte{}
			v.x.WriteToArray32(&dest)
			data = append(data, dest[:])
		default:
			panic(vm.ToValue("Unsupported type for Keccak256"))
		}
	}
	return vm.ToValue(vm.NewArrayBuffer(crypto.Keccak256(data...)))
}

func Sha256(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	h := sha256.New()
	for _, arg := range(f.Arguments) {
		switch v := arg.Export().(type) {
		case goja.ArrayBuffer:
			h.Write(v.Bytes())
		case Uint256:
			dest := [32]byte{}
			v.x.WriteToArray32(&dest)
			h.Write(dest[:])
		default:
			panic(vm.ToValue("Unsupported type for Keccak256"))
		}
	}
	return vm.ToValue(vm.NewArrayBuffer(h.Sum(nil)))
}

func BufConcat(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	data := [][]byte{}
	totalLen := 0
	for _, arg := range(f.Arguments) {
		switch v := arg.Export().(type) {
		case goja.ArrayBuffer:
			data = append(data, v.Bytes())
			totalLen += len(v.Bytes())
		default:
			panic(vm.ToValue("Unsupported type for BufConcat"))
		}
	}
	result := make([]byte, totalLen, 0)
	for _, bz := range(data) {
		result = append(result, bz...)
	}
	return vm.ToValue(vm.NewArrayBuffer(result))
}

func B64ToBuf(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 1 {
		panic(IncorrectArgumentCount)
	}
	str, ok := f.Arguments[0].Export().(string)
	if !ok {
		panic(goja.NewSymbol("The first argument must be string"))
	}
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		panic(goja.NewSymbol("error in B64ToBuf: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(data))
}

func HexToBuf(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 1 {
		panic(IncorrectArgumentCount)
	}
	str, ok := f.Arguments[0].Export().(string)
	if !ok {
		panic(goja.NewSymbol("The first argument must be string"))
	}
	data, err := hex.DecodeString(str)
	if err != nil {
		panic(goja.NewSymbol("error in HexToBuf: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(data))
}

func BufToB64(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	buf := getOneArrayBuffer(f)
	str := base64.StdEncoding.EncodeToString(buf)
	return vm.ToValue(str)
}

func BufToHex(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	buf := getOneArrayBuffer(f)
	str := hex.EncodeToString(buf)
	return vm.ToValue(str)
}

func BufEqual(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	a, b := getTwoArrayBuffers(f)
	return vm.ToValue(bytes.Equal(a, b))
}

func BufCompare(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	a, b := getTwoArrayBuffers(f)
	return vm.ToValue(bytes.Compare(a, b))
}

func VerifySignature(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	pubkey, digestHash, signature := getThreeArrayBuffers(f)
	result := crypto.VerifySignature(pubkey, digestHash, signature)
	return vm.ToValue(result)
}

func Ecrecover(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	hash, sig := getTwoArrayBuffers(f)
	bz, err := crypto.Ecrecover(hash, sig)
	if err != nil {
		panic(goja.NewSymbol("error in Ecrecover: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

