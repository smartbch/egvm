package extension

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	SerializeMapsScriptTemplate = `
		let m = NewOrderedIntMap()
		m.Set('a', 1)
		m.Set('b', 2)
		m.Set('c', 3)

		const bz = SerializeMaps(m)

		let ms = DeserializeMaps(bz)
		let m2 = ms[0]
		const [v1, ok1] = m2.Get('a')
		const len1 = m2.Len()
	`
)

func setupGojaVmForCodec() *goja.Runtime {
	vm := goja.New()
	vm.Set("SerializeMaps", SerializeMaps)
	vm.Set("DeserializeMaps", DeserializeMaps)
	vm.Set("NewOrderedIntMap", NewOrderedIntMap)
	return vm
}

// FIXME: DeserializeMaps
func TestSerial(t *testing.T) {
	vm := setupGojaVmForCodec()
	_, err := vm.RunString(SerializeMapsScriptTemplate)
	require.NoError(t, err)

	bz := vm.Get("bz").Export().(goja.ArrayBuffer)
	fmt.Printf("bz: %v\n", bz.Bytes())

	ok1 := vm.Get("ok1").Export().(bool)
	v1 := vm.Get("v1").Export().(int64)
	len1 := vm.Get("len1").Export().(int64)
	require.False(t, ok1)
	require.EqualValues(t, 1, v1)
	require.EqualValues(t, 3, len1)
}
