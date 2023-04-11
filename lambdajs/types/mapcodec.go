package types

import (
	"github.com/dop251/goja"
	"github.com/tinylib/msgp/msgp"

	"github.com/smartbch/pureauth/lambdajs/utils"
)

const (
	OrderedIntMapTag = byte(0)
	OrderedStrMapTag = byte(1)
	OrderedBufMapTag = byte(2)
)

// arguments: maps ...interface{}
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

// Note: Each call deserializes only one map and returns the rest of the array buffer
// arguments: bz goja.ArrayBuffer, tag uint8
func DeserializeMap(f goja.FunctionCall, vm *goja.Runtime) goja.Value {
	if len(f.Arguments) != 2 {
		panic(utils.IncorrectArgumentCount)
	}

	buf, ok := f.Arguments[0].Export().(goja.ArrayBuffer)
	if !ok {
		panic(goja.NewSymbol("The first argument must be ArrayBuffer"))
	}

	b := buf.Bytes()
	if len(b) == 0 {
		panic(goja.NewSymbol("Empty map bytes"))
	}

	expectedTag, ok := f.Arguments[1].Export().(int64)
	if !ok {
		panic(goja.NewSymbol("The second argument must be uint8"))
	}

	if expectedTag < 0 || expectedTag > int64(OrderedBufMapTag) {
		panic(goja.NewSymbol("Invalid map tag"))
	}

	var mv goja.Value
	tag, b, err := msgp.ReadByteBytes(b)
	if err != nil || tag > OrderedBufMapTag {
		panic(goja.NewSymbol("Tag byte error in DeserializeMaps " + err.Error()))
	}

	if tag != byte(expectedTag) {
		panic(goja.NewSymbol("Tag byte is not equal to inputted type"))
	}

	if tag == OrderedIntMapTag {
		im := NewOrderedIntMap()
		b, err = im.loadFrom(b)
		mv = vm.ToValue(im)
	} else if tag == OrderedStrMapTag {
		sm := NewOrderedStrMap()
		b, err = sm.loadFrom(b)
		mv = vm.ToValue(sm)
	} else if tag == OrderedBufMapTag {
		bm := NewOrderedBufMap()
		b, err = bm.loadFrom(b)
		mv = vm.ToValue(bm)
	}

	if err != nil {
		panic(goja.NewSymbol("Error in loading map"))
	}

	var result [2]any
	result = [2]any{mv, vm.NewArrayBuffer(b)}
	return vm.ToValue(result)
}
