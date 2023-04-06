package extension

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"io"
	"fmt"
	"hash"
	"strings"
	"unicode"

	"github.com/cespare/xxhash/v2"
	"github.com/dop251/goja"
	ecies "github.com/ecies/go/v2"
	"github.com/gcash/bchutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/klauspost/compress/zstd"
	"github.com/tyler-smith/go-bip32"
	"github.com/vechain/go-ecvrf"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"

	"github.com/smartbch/pureauth/keygrantor"
)

func RegisterFunctions(vm *goja.Runtime) {
	vm.Set("HexToU256", HexToU256)
	vm.Set("BufToU256", BufToU256)
	vm.Set("HexToS256", HexToS256)
	vm.Set("BufToS256", BufToS256)
	vm.Set("U256", U256)
	vm.Set("S256", S256)
	vm.Set("AesGcmDecrypt", AesGcmDecrypt)
	vm.Set("AesGcmEncrypt", AesGcmEncrypt)
	vm.Set("BufToPrivateKey", BufToPrivateKey)
	vm.Set("BufToPublicKey", BufToPublicKey)
	vm.Set("Keccak256", Keccak256)
	vm.Set("Sha256", Sha256)
	vm.Set("Ripemd160", Ripemd160)
	vm.Set("BufConcat", BufConcat)
	vm.Set("HexToBuf", HexToBuf)
	vm.Set("B64ToBuf", B64ToBuf)
	vm.Set("BufToB64", BufToB64)
	vm.Set("BufToHex", BufToHex)
	vm.Set("BufEqual", BufEqual)
	vm.Set("BufCompare", BufCompare)
	vm.Set("VerifySignature", VerifySignature)
	vm.Set("Ecrecover", Ecrecover)
	vm.Set("ZstdCompress", ZstdCompress)
	vm.Set("ZstdDecompress", ZstdDecompress)
	vm.Set("VerifyMerkleProofSha256", VerifyMerkleProofSha256)
	vm.Set("VerifyMerkleProofKeccak256", VerifyMerkleProofKeccak256)
	vm.Set("HttpRequest", HttpRequest)
	vm.Set("SignTxAndSerialize", SignTxAndSerialize)
	vm.Set("ParseTxInHex", ParseTxInHex)
}

var (
	IncorrectArgumentCount = goja.NewSymbol("incorrect argument count")
)

func getOneUint64(f goja.FunctionCall) uint64 {
	if len(f.Arguments) != 1 {
		panic(IncorrectArgumentCount)
	}
	a, ok := f.Arguments[0].Export().(uint64)
	if !ok {
		panic(goja.NewSymbol("The first argument must be number"))
	}
	return a
}

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
		return vm.ToValue([2]any{nil, false})
	}
	return vm.ToValue([2]any{vm.NewArrayBuffer(bz), true})
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

func (prv PrivateKey) toECDSA() *ecdsa.PrivateKey {
	return &ecdsa.PrivateKey {
		D: prv.key.D,
		PublicKey: ecdsa.PublicKey {
			Curve: prv.key.PublicKey.Curve,
			X: prv.key.PublicKey.X,
			Y: prv.key.PublicKey.Y,
		},
	}
}

func (prv PrivateKey) Sign(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	msg := getOneArrayBuffer(f)
	ecdsaPrv := prv.toECDSA()
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

func (prv PrivateKey) Serialize(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
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

func (pub PublicKey) toBuf(f goja.FunctionCall, vm *goja.Runtime, compressed bool) goja.Value {
	if len(f.Arguments) != 1 {
		panic(IncorrectArgumentCount)
	}
	compressed, ok := f.Arguments[0].Export().(bool)
	if !ok {
		panic(goja.NewSymbol("The first argument must be boolean"))
	}
	return vm.ToValue(vm.NewArrayBuffer(pub.key.Bytes(compressed)))
}

func (pub PublicKey) SerializeCompressed(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return pub.toBuf(f, vm, true)
}

func (pub PublicKey) SerializeUncompressed(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return pub.toBuf(f, vm, false)
}

func (prv PrivateKey) Decrypt(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	buf := getOneArrayBuffer(f)
	bz, err := ecies.Decrypt(prv.key, buf)
	if err != nil {
		return vm.ToValue([2]any{nil, false})
	}
	return vm.ToValue([2]any{vm.NewArrayBuffer(bz), true})
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

func (pub PublicKey) toECDSA() *ecdsa.PublicKey {
	return &ecdsa.PublicKey {
		Curve: pub.key.Curve,
		X: pub.key.X,
		Y: pub.key.Y,
	}
}

func (pub PublicKey) ToEvmAddress(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(IncorrectArgumentCount)
	}
	key := pub.toECDSA()
	addr := crypto.PubkeyToAddress(*key)
	return vm.ToValue(vm.NewArrayBuffer(addr[:]))
}

func (pub PublicKey) ToCashAddress(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(IncorrectArgumentCount)
	}
	pubKeyHash := bchutil.Hash160(pub.key.Bytes(true))
	return vm.ToValue(vm.NewArrayBuffer(pubKeyHash[:]))
}

func (prv PrivateKey) VrfProve(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	alpha := getOneArrayBuffer(f)
	beta, pi, err := ecvrf.Secp256k1Sha256Tai.Prove(prv.toECDSA(), alpha)
	if err != nil {
		panic(goja.NewSymbol("error in VrfProve: "+err.Error()))
	}
	return vm.ToValue([2]goja.ArrayBuffer{vm.NewArrayBuffer(beta), vm.NewArrayBuffer(pi)})
}

func (pub PublicKey) VrfVerify(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	alpha, pi := getTwoArrayBuffers(f)
	beta, err := ecvrf.Secp256k1Sha256Tai.Verify(pub.toECDSA(), alpha, pi)
	if err != nil {
		panic(goja.NewSymbol("error in VrfVerify: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(beta[:]))
}

// ===============

type Bip32Key struct {
	key *bip32.Key
}

func B58ToBip32Key(data string) Bip32Key {
	key, err := bip32.B58Deserialize(data)
	if err != nil {
		panic(goja.NewSymbol("error in B58ToBip32Key: "+err.Error()))
	}
	return Bip32Key{key: key}
}

func BufToBip32Key(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	buf := getOneArrayBuffer(f)
	key, err := bip32.Deserialize(buf)
	if err != nil {
		panic(goja.NewSymbol("error in BufToBip32Key: "+err.Error()))
	}
	return vm.ToValue(Bip32Key{key: key})
}

func GenerateRandomBip32Key() Bip32Key {
	return Bip32Key{key: keygrantor.GetRandomExtPrivKey()}
}

func (key Bip32Key) B58Serialize() string {
	return key.key.B58Serialize()
}

func (key Bip32Key) NewChildKey(childIdx uint32) Bip32Key {
	k, err := key.key.NewChildKey(childIdx)
	if err != nil {
		panic(goja.NewSymbol("error in NewChildKey: "+err.Error()))
	}
	return Bip32Key{key: k}
}

func (key Bip32Key) PublicKey() Bip32Key {
	return Bip32Key{key: key.key.PublicKey()}
}

func (key Bip32Key) Serialize(f goja.FunctionCall, vm *goja.Runtime) goja.Value { 
	bz, _ := key.key.Serialize() // impossible to generate error
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

func (key Bip32Key) IsPrivate() bool { 
	return key.key.IsPrivate
}

func (key Bip32Key) ToPrivateKey() PrivateKey { 
	if !key.IsPrivate() {
		panic(goja.NewSymbol("It's not private"))
	}
	return PrivateKey{key: ecies.NewPrivateKeyFromBytes(key.key.Key)}
}

func (key Bip32Key) DeriveWithBytes32(f goja.FunctionCall, vm *goja.Runtime) goja.Value { 
	var hash [32]byte
	bz := getOneArrayBuffer(f)
	copy(hash[:], bz)
	bip32Key := keygrantor.DeriveKey(key.key, hash)
	return vm.ToValue(Bip32Key{key: bip32Key})
}
	
func (key Bip32Key) Derive(f goja.FunctionCall, vm *goja.Runtime) goja.Value { 
	child := key.key
	for i, arg := range(f.Arguments) {
		n, ok := arg.Export().(uint32)
		if !ok {
			panic(goja.NewSymbol(fmt.Sprintf("The argument #%d is not uint32", i)))
		}
		var err error
		child, err = child.NewChildKey(n)
		if err != nil {
			panic(vm.ToValue("Error in Derive: "+err.Error()))
		}
	}
	return vm.ToValue(Bip32Key{key: child})
}

// ===============

func isASCII(s string) bool {
	for _, c := range s {
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func hashFunc(f goja.FunctionCall, vm *goja.Runtime, h io.Writer) {
	var buf [32]byte
	for _, arg := range(f.Arguments) {
		switch v := arg.Export().(type) {
		case string:
			if !isASCII(v) {
				panic(vm.ToValue("Non-ascii string is not supported for hash"))
			}
			h.Write([]byte(v))
		case goja.ArrayBuffer:
			h.Write(v.Bytes())
		case Uint256:
			v.x.WriteToArray32(&buf)
			h.Write(buf[:])
		default:
			panic(vm.ToValue("Unsupported type for hash"))
		}
	}
}

func Keccak256(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	h := sha3.NewLegacyKeccak256()
	hashFunc(f, vm, h)
	return vm.ToValue(vm.NewArrayBuffer(h.Sum(nil)))
}

func Sha256(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	h := sha256.New()
	hashFunc(f, vm, h)
	return vm.ToValue(vm.NewArrayBuffer(h.Sum(nil)))
}

func Ripemd160(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	h := ripemd160.New()
	hashFunc(f, vm, h)
	return vm.ToValue(vm.NewArrayBuffer(h.Sum(nil)))
}

func XxHash(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	h := xxhash.New()
	hashFunc(f, vm, h)
	return vm.ToValue(h.Sum64())
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

func BufReverse(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	a := getOneArrayBuffer(f)
	b := make([]byte, 0, len(a))
	for i := range a {
		b = append(b, a[len(a)-1-i])
	}
	return vm.ToValue(vm.NewArrayBuffer(a))
}

type BufBuilder struct {
	b *strings.Builder
}

func (b BufBuilder) Len() int {
	return b.b.Len()
}

func (b BufBuilder) Reset() {
	b.b.Reset()
}

func (b BufBuilder) Write(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	bz := getOneArrayBuffer(f)
	n, err := b.b.Write(bz)
	if err != nil {
		panic(goja.NewSymbol("error in Ecrecover: "+err.Error()))
	}
	return vm.ToValue(n)
}

func (b BufBuilder) ToBuf(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(IncorrectArgumentCount)
	}
	return vm.ToValue(vm.NewArrayBuffer([]byte(b.b.String())))
}

func BufToU64BE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	bz := getOneArrayBuffer(f)
	return vm.ToValue(binary.BigEndian.Uint64(bz))
}

func BufToU64LE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	bz := getOneArrayBuffer(f)
	return vm.ToValue(binary.LittleEndian.Uint64(bz))
}

func BufToU32BE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	bz := getOneArrayBuffer(f)
	return vm.ToValue(binary.BigEndian.Uint32(bz))
}

func BufToU32LE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	bz := getOneArrayBuffer(f)
	return vm.ToValue(binary.LittleEndian.Uint32(bz))
}

func U64ToBufBE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u64 := getOneUint64(f)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], u64)
	return vm.ToValue(vm.NewArrayBuffer(buf[:]))
}

func U64ToBufLE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u64 := getOneUint64(f)
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], u64)
	return vm.ToValue(vm.NewArrayBuffer(buf[:]))
}

func U32ToBufBE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u64 := getOneUint64(f)
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(u64))
	return vm.ToValue(vm.NewArrayBuffer(buf[:]))
}

func U32ToBufLE(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u64 := getOneUint64(f)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(u64))
	return vm.ToValue(vm.NewArrayBuffer(buf[:]))
}

// ================================

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

// ================================

func ZstdDecompress(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	src := getOneArrayBuffer(f)
	var decoder, _ = zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	bz, err := decoder.DecodeAll(src, nil)
	if err != nil {
		panic(goja.NewSymbol("error in ZstdDecompress: "+err.Error()))
	}
	return vm.ToValue(vm.NewArrayBuffer(bz))
}

func ZstdCompress(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	src := getOneArrayBuffer(f)
	var encoder, _ = zstd.NewWriter(nil)
	bz := encoder.EncodeAll(src, make([]byte, 0, len(src)))
	return vm.ToValue(vm.NewArrayBuffer(bz))
} 

// ================================

func verifyMerkleProof(f goja.FunctionCall, vm *goja.Runtime, h hash.Hash) goja.Value {
	root, proof, leaf := getThreeArrayBuffers(f)
	if len(leaf) != 32 {
		panic(goja.NewSymbol(fmt.Sprintf("Invalid merkle tree leaf size %d", len(leaf))))
	}
	if len(proof) % 32 != 0 {
		panic(goja.NewSymbol(fmt.Sprintf("Invalid merkle proof size %d", len(proof))))
	}
	computedHash := leaf
	for offset := 0; offset < len(proof); offset += 32 {
		h.Reset()
		node := proof[offset:offset+32]
		if bytes.Compare(computedHash, node) < 0 {
			h.Write(computedHash)
			h.Write(node)
		} else {
			h.Write(node)
			h.Write(computedHash)
		}
		computedHash = h.Sum(nil)
	}
	ok := bytes.Equal(root, computedHash)
	return vm.ToValue(ok)
}

func VerifyMerkleProofSha256(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return verifyMerkleProof(f, vm, sha256.New())
}

func VerifyMerkleProofKeccak256(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return verifyMerkleProof(f, vm, sha3.NewLegacyKeccak256())
}

// =================== non-deterministic =============

func ND_ReadTsc() uint64 {
	return 0 //TODO
}

func ND_GetEphemeralID() string {
	return "" //TODO
}



