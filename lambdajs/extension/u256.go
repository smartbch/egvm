package extension

import (
	"github.com/dop251/goja"
	"github.com/holiman/uint256"
)

const (
	MAX_SAFE_INTEGER = (uint64(1) << 53) - 1
)

type Uint256 struct {
	x *uint256.Int
}

func HexToU256(hex string) Uint256 {
	x, err := uint256.FromHex(hex)
	if err != nil {
		panic(goja.NewSymbol("invalid hex string"))
	}
	return Uint256{x: x}
}

func BufToU256(buf goja.ArrayBuffer) Uint256 {
	return Uint256{x: uint256.NewInt(0).SetBytes(buf.Bytes())}
}

func U256(v uint64) Uint256 {
	if v > MAX_SAFE_INTEGER {
		panic(goja.NewSymbol("larger than Number.MAX_SAFE_INTEGER"))
	}
	return Uint256{x: uint256.NewInt(v)}
}

func (u Uint256) ToS256() Sint256 {
	return Sint256{x: u.x.Clone()}
}

func (u Uint256) Add(v Uint256) Uint256 {
	result, overflow := uint256.NewInt(0).AddOverflow(u.x, v.x)
	if overflow {
		panic(goja.NewSymbol("overflow in addition"))
	}
	return Uint256{x: result}
}

func (u Uint256) UnsafeAdd(v Uint256) Uint256 {
	result := uint256.NewInt(0).Add(u.x, v.x)
	return Uint256{x: result}
}

func (u Uint256) And(v Uint256) Uint256 {
	result := uint256.NewInt(0).And(u.x, v.x)
	return Uint256{x: result}
}

func (u Uint256) Div(v Uint256) Uint256 {
	if v.x.IsZero() {
		panic(goja.NewSymbol("divide by zero"))
	}
	result := uint256.NewInt(0).Div(u.x, v.x)
	return Uint256{x: result}
}

func (u Uint256) Mod(v Uint256) Uint256 {
	if v.x.IsZero() {
		panic(goja.NewSymbol("divide by zero"))
	}
	result := uint256.NewInt(0).Mod(u.x, v.x)
	return Uint256{x: result}
}

func (u Uint256) DivMod(v Uint256) [2]Uint256 {
	if v.x.IsZero() {
		panic(goja.NewSymbol("divide by zero"))
	}
	quo, rem := uint256.NewInt(0).DivMod(u.x, v.x, uint256.NewInt(0))
	return [2]Uint256{{x: quo}, {x: rem}}
}

func (u Uint256) Exp(v Uint256) Uint256 {
	result := uint256.NewInt(0).Exp(u.x, v.x)
	return Uint256{x: result}
}

func (u Uint256) Gt(v Uint256) bool {
	return u.x.Gt(v.x)
}

func (u Uint256) Gte(v Uint256) bool {
	return !v.x.Gt(u.x)
}

func (u Uint256) GtNum(v uint64) bool {
	if v > MAX_SAFE_INTEGER {
		panic(goja.NewSymbol("larger than Number.MAX_SAFE_INTEGER"))
	}
	return u.x.GtUint64(v)
}

func (u Uint256) GteNum(v uint64) bool {
	if v > MAX_SAFE_INTEGER {
		panic(goja.NewSymbol("larger than Number.MAX_SAFE_INTEGER"))
	}
	return !U256(v).x.Gt(u.x)
}

func (u Uint256) IsZero() bool {
	return u.x.IsZero()
}

func (u Uint256) Equal(v Uint256) bool {
	return u.x.Eq(v.x)
}

func (u Uint256) Lt(v Uint256) bool {
	return u.x.Lt(v.x)
}

func (u Uint256) Lte(v Uint256) bool {
	return !v.x.Lt(u.x)
}

func (u Uint256) LtNum(v uint64) bool {
	if v > MAX_SAFE_INTEGER {
		panic(goja.NewSymbol("larger than Number.MAX_SAFE_INTEGER"))
	}
	return u.x.LtUint64(v)
}

func (u Uint256) LteNum(v uint64) bool {
	if v > MAX_SAFE_INTEGER {
		panic(goja.NewSymbol("larger than Number.MAX_SAFE_INTEGER"))
	}
	return !U256(v).x.Lt(u.x)
}

func (u Uint256) Mul(v Uint256) Uint256 {
	result, overflow := uint256.NewInt(0).MulOverflow(u.x, v.x)
	if overflow {
		panic(goja.NewSymbol("overflow in multiplication"))
	}
	return Uint256{x: result}
}

func (u Uint256) UnsafeMul(v Uint256) Uint256 {
	result := uint256.NewInt(0).Mul(u.x, v.x)
	return Uint256{x: result}
}

func (u Uint256) Not() Uint256 {
	result := uint256.NewInt(0).Not(u.x)
	return Uint256{x: result}
}

func (u Uint256) Or(v Uint256) Uint256 {
	result := uint256.NewInt(0).Or(u.x, v.x)
	return Uint256{x: result}
}

func (u Uint256) Lsh(v uint) Uint256 {
	result := uint256.NewInt(0).Lsh(u.x, v)
	return Uint256{x: result}
}

func (u Uint256) Rsh(v uint) Uint256 {
	result := uint256.NewInt(0).Rsh(u.x, v)
	return Uint256{x: result}
}

func (u Uint256) Sqrt() Uint256 {
	result := uint256.NewInt(0).Sqrt(u.x)
	return Uint256{x: result}
}

func (u Uint256) Sub(v Uint256) Uint256 {
	if u.x.Lt(v.x) {
		panic(goja.NewSymbol("Overflow in substraction"))
	}
	result := uint256.NewInt(0).Sub(u.x, v.x)
	return Uint256{x: result}
}

func (u Uint256) UnsafeSub(v Uint256) Uint256 {
	result := uint256.NewInt(0).Sub(u.x, v.x)
	return Uint256{x: result}
}

func (u Uint256) IsSafeInteger() bool {
	u64, overflow := u.x.Uint64WithOverflow()
	return u64 <= MAX_SAFE_INTEGER && !overflow
}

func (u Uint256) ToSafeInteger() uint64 {
	u64, overflow := u.x.Uint64WithOverflow()
	safe := u64 <= MAX_SAFE_INTEGER && !overflow
	if !safe {
		panic(goja.NewSymbol("Overflow in ToSafeInteger"))
	}
	return u64
}

func (u Uint256) ToBuf(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(vm.ToValue("ToBuf has no arguments."))
	}
	var dest [32]byte
	u.x.WriteToArray32(&dest)
	return vm.ToValue(vm.NewArrayBuffer(dest[:]))
}

func (u Uint256) ToHex(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(vm.ToValue("ToHex has no arguments."))
	}
	return vm.ToValue(u.x.Hex())
}

func (u Uint256) ToString(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 0 {
		panic(vm.ToValue("ToHex has no arguments."))
	}
	return vm.ToValue(u.x.String())
}
