package gojs

import (
	"crypto/ecdsa"

	"github.com/dop251/goja"
	ecies "github.com/ecies/go/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

type Uint256 struct {
	x *uint256.Int
}

func Uint256FromHex(hex string) Uint256 {
	x, _ := uint256.FromHex(hex) // TODO error handling
	return Uint256{x: x}
}

func Uint256FromNumber(n uint64) Uint256 {
	return Uint256{x: uint256.NewInt(n)}
}

func (u Uint256) Add(v Uint256) Uint256 {
	return Uint256{
		x: uint256.NewInt(0).Add(u.x, v.x),
	}
}

func (u Uint256) Sub(v Uint256) Uint256 {
	return Uint256{
		x: uint256.NewInt(0).Sub(u.x, v.x),
	}
}

func Uint256FromBuffer(buf goja.ArrayBuffer) Uint256 {
	return Uint256{x: uint256.NewInt(0).SetBytes(buf.Bytes())}
}

func Uint256ToBuffer(f goja.FunctionCall, rt *goja.Runtime) goja.Value {
	x, _ := f.Argument(0).Export().(Uint256) // TODO error check
	var dest [32]byte
	x.x.WriteToArray32(&dest)
	return rt.ToValue(rt.NewArrayBuffer(dest[:]))
}

// =============================

type PrivateKey struct {
	key *ecies.PrivateKey
}

type PublicKey struct {
	key *ecies.PublicKey
}

func PrivateKeyFromBytes(buf goja.ArrayBuffer) PrivateKey {
	key := ecies.NewPrivateKeyFromBytes(buf.Bytes())
	return PrivateKey{key: key}
}

func PublicKeyFromBytes(buf goja.ArrayBuffer) PublicKey {
	key, _ := ecies.NewPublicKeyFromBytes(buf.Bytes())
	return PublicKey{key: key}
}

func (k PrivateKey) GetPublicKey() PublicKey {
	return PublicKey{key: k.key.PublicKey}
}

func Encapsulate(f goja.FunctionCall, rt *goja.Runtime) goja.Value {
	prv, _ := f.Argument(0).Export().(PrivateKey)
	pub, _ := f.Argument(1).Export().(PublicKey)
	bz, _ := prv.key.Encapsulate(pub.key)
	return rt.ToValue(rt.NewArrayBuffer(bz))
}

func Sign(f goja.FunctionCall, rt *goja.Runtime) goja.Value {
	prv, _ := f.Argument(0).Export().(PrivateKey)
	buf, _ := f.Argument(1).Export().(goja.ArrayBuffer)
	ecdsaPrv := &ecdsa.PrivateKey {
		D: prv.key.D,
		PublicKey: ecdsa.PublicKey {
			Curve: prv.key.PublicKey.Curve,
			X: prv.key.PublicKey.X,
			Y: prv.key.PublicKey.Y,
		},
	}
	sig, _ := crypto.Sign(buf.Bytes(), ecdsaPrv)
	return rt.ToValue(rt.NewArrayBuffer(sig))
}

