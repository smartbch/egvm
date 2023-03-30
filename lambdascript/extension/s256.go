package extension

import (
	"github.com/dop251/goja"
	"github.com/holiman/uint256"
)

type Sint256 struct {
	x *uint256.Int
}

func HexToS256(hex string) Sint256 {
	x, err := uint256.FromHex(hex)
	if err != nil {
		panic(goja.NewSymbol("invalid hex string"))
	}
	return Sint256{x: x}
}

func BufToS256(buf goja.ArrayBuffer) Sint256 {
	return Sint256{x: uint256.NewInt(0).SetBytes(buf.Bytes())}
}

func S256(n int64) Sint256 {
	if n>=0 {
		return Sint256{x: uint256.NewInt(uint64(n))}
	}
	tmp := uint256.NewInt(uint64(-n))
	return Sint256{x: uint256.NewInt(0).Neg(tmp)}
}

func (u Sint256) ToU256() Uint256 {
	return Uint256{x: u.x.Clone()}
}

func (u Sint256) Abs() Sint256 {
	result := uint256.NewInt(0).Abs(u.x)
	return Sint256{x: result}
}

func (u Sint256) Neg() Sint256 {
	result := uint256.NewInt(0).Neg(u.x)
	return Sint256{x: result}
}

func (u Sint256) Add(v Sint256) Sint256 {
	errSymbol := goja.NewSymbol("overflow in signed addition")
	result, overflow := (*uint256.Int)(nil), false
	if u.x.Sign() >= 0 && v.x.Sign() >= 0 {
		result, overflow = uint256.NewInt(0).AddOverflow(u.x, v.x)
	} else {
		result = uint256.NewInt(0).Add(u.x, v.x)
		overflow = u.x.Sign() < 0 && v.x.Sign() < 0 && result.Sign() >= 0
	}
	if overflow {
		panic(errSymbol)
	}
	return Sint256{x: result}
}

func (u Sint256) Equal(v Sint256) bool {
	return u.x.Eq(v.x)
}

func (u Sint256) IsZero() bool {
	return u.x.IsZero()
}

func (u Sint256) Sign() int {
	return u.x.Sign()
}

func (u Sint256) Sub(v Sint256) Sint256 {
	errSymbol := goja.NewSymbol("overflow in signed substraction")
	result := uint256.NewInt(0).Sub(u.x, v.x)
	uSign, vSign, rSign := u.x.Sign(), v.x.Sign(), result.Sign()
	overflow := (uSign > 0 && vSign < 0 && rSign < 0) || // pos - neg < 0
		(uSign < 0 && vSign > 0 && rSign > 0) // neg - pos > 0
	if overflow {
		panic(errSymbol)
	}
	return Sint256{x: result}
}

func (u Sint256) Mul(v Sint256) Sint256 {
	p, q := uint256.NewInt(0).Abs(u.x), uint256.NewInt(0).Abs(v.x)
	result, overflow := uint256.NewInt(0).MulOverflow(p, q)
	if overflow {
		panic(goja.NewSymbol("overflow in signed multiplication"))
	}
	if u.x.Sign() * v.x.Sign() == -1 {
		result = uint256.NewInt(0).Neg(result)
	}
	return Sint256{x: result}
}

func (u Sint256) Div(v Sint256) Sint256 {
	if v.x.IsZero() {
		panic(goja.NewSymbol("divide by zero"))
	}
	p, q := uint256.NewInt(0).Abs(u.x), uint256.NewInt(0).Abs(v.x)
	result := uint256.NewInt(0).Div(p, q)
	if u.x.Sign() * v.x.Sign() == -1 {
		result = uint256.NewInt(0).Neg(result)
	}
	return Sint256{x: result}
}

func (u Sint256) Lsh(v uint) Sint256 {
	result := uint256.NewInt(0).Lsh(u.x, v)
	return Sint256{x: result}
}

func (u Sint256) Rsh(v uint) Sint256 {
	result := uint256.NewInt(0).SRsh(u.x, v)
	return Sint256{x: result}
}

func (u Sint256) Gt(v Sint256) bool {
	return u.x.Gt(v.x)
}

func (u Sint256) Gte(v Sint256) bool {
	return !v.x.Gt(u.x)
}

func (u Sint256) GtNum(v int64) bool {
	if v > int64(MAX_SAFE_INTEGER) || -v > int64(MAX_SAFE_INTEGER) {
		panic(goja.NewSymbol("larger than Number.MAX_SAFE_INTEGER"))
	}
	return u.x.Gt(S256(v).x)
}

func (u Sint256) GteNum(v int64) bool {
	if v > int64(MAX_SAFE_INTEGER) || -v > int64(MAX_SAFE_INTEGER) {
		panic(goja.NewSymbol("larger than Number.MAX_SAFE_INTEGER"))
	}
	return !S256(v).x.Gt(u.x)
}

func (u Sint256) Lt(v Sint256) bool {
	return u.x.Lt(v.x)
}

func (u Sint256) Lte(v Sint256) bool {
	return !v.x.Lt(u.x)
}

func (u Sint256) LtNum(v int64) bool {
	if v > int64(MAX_SAFE_INTEGER) || -v > int64(MAX_SAFE_INTEGER) {
		panic(goja.NewSymbol("larger than Number.MAX_SAFE_INTEGER"))
	}
	return u.x.Lt(S256(v).x)
}

func (u Sint256) LteNum(v int64) bool {
	if v > int64(MAX_SAFE_INTEGER) {
		panic(goja.NewSymbol("larger than Number.MAX_SAFE_INTEGER"))
	}
	return !S256(v).x.Lt(u.x)
}

func (u Sint256) IsSafeInteger() bool {
	u64, overflow := uint64(0), false
	if u.x.Sign() < 0 {
		y := uint256.NewInt(0).Neg(u.x)
		u64, overflow = y.Uint64WithOverflow()
	}
	u64, overflow = u.x.Uint64WithOverflow()
	return u64 <= MAX_SAFE_INTEGER && !overflow 
}

func (u Sint256) ToSafeInteger() int64 {
	u64, overflow := uint64(0), false
	if u.x.Sign() < 0 {
		y := uint256.NewInt(0).Neg(u.x)
		u64, overflow = y.Uint64WithOverflow()
	}
	u64, overflow = u.x.Uint64WithOverflow()
	safe := u64 <= MAX_SAFE_INTEGER && !overflow 
	if !safe {
		panic(goja.NewSymbol("Overflow in ToSafeInteger"))
	}
	if u.x.Sign() < 0 {
		return -int64(u64)
	}
	return int64(u64)
}

func (u Sint256) ToBuf(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(vm.ToValue("ToBuf has no arguments."))
	}
	var dest [32]byte
	u.x.WriteToArray32(&dest)
	return vm.ToValue(vm.NewArrayBuffer(dest[:]))
}

func (u Sint256) ToHex(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(vm.ToValue("ToHex has no arguments."))
	}
	return vm.ToValue(u.x.Hex())
}

func (u Sint256) ToString(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(vm.ToValue("ToHex has no arguments."))
	}
	if u.x.Sign() < 0 {
		return vm.ToValue("-"+uint256.NewInt(0).Neg(u.x).String())
	}
	return vm.ToValue(u.x.String())
}
