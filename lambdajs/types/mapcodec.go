package types

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/tinylib/msgp/msgp"

	"github.com/smartbch/pureauth/lambdajs/utils"
)

const (
	OrderedIntMapTag = byte(0)
	OrderedStrMapTag = byte(1)
	OrderedBufMapTag = byte(2)
)

func SerializeMaps(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	totalSize := 0
	for _, arg := range f.Arguments {
		switch v := arg.Export().(type) {
		case OrderedIntMap:
			totalSize += 2 + v.estimatedSize
		case OrderedStrMap:
			totalSize += 2 + v.estimatedSize
		case OrderedBufMap:
			totalSize += 2 + v.estimatedSize
		default:
			panic(vm.ToValue("Unsupported type for EncodeMaps"))
		}
	}

	// tag + len + size
	b := make([]byte, 0, totalSize)
	for _, arg := range f.Arguments {
		switch v := arg.Export().(type) {
		case OrderedIntMap:
			b = msgp.AppendByte(b, OrderedIntMapTag)
			b = v.dumpTo(b)
		case OrderedStrMap:
			b = msgp.AppendByte(b, OrderedStrMapTag)
			b = v.dumpTo(b)
		case OrderedBufMap:
			b = msgp.AppendByte(b, OrderedBufMapTag)
			b = v.dumpTo(b)
		}
	}
	return vm.ToValue(vm.NewArrayBuffer(b))
}

func DeserializeMaps(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	b := utils.GetOneArrayBuffer(f)
	if len(b) == 0 {
		panic(goja.NewSymbol("Empty map bytes"))
	}

	var result []any
	var tag byte
	var err error
	var remainBz []byte
	copy(remainBz, b)
	for i := 0; len(remainBz) != 0; i++ {
		tag, remainBz, err = msgp.ReadByteBytes(b)
		if err != nil || tag > OrderedBufMapTag {
			panic(goja.NewSymbol("Tag byte error in DeserializeMaps " + err.Error()))
		}

		if tag == OrderedIntMapTag {
			m := NewOrderedIntMap()
			b, err = m.loadFrom(b)
			result = append(result, m)
		} else if tag == OrderedStrMapTag {
			m := NewOrderedStrMap()
			b, err = m.loadFrom(b)
			result = append(result, m)
		} else if tag == OrderedBufMapTag {
			m := NewOrderedBufMap()
			b, err = m.loadFrom(b)
			result = append(result, m)
		}

		if err != nil {
			panic(goja.NewSymbol(fmt.Sprintf("Error in loading #%d map", i)))
		}
	}
	return vm.ToValue(result)
}
