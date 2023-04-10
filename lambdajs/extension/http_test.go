package extension

import (
	"fmt"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

const (
	HttpScriptTemplate = `
		const resp = HttpRequest('GET', 'https://elfinauth.paralinker.io/smartbch/eh_ping', '')
	`
)

func setupGojaVmForHttp() *goja.Runtime {
	vm := goja.New()
	vm.Set("HttpRequest", HttpRequest)
	return vm
}

// TODO: add more http test
func TestHttpRequest(t *testing.T) {
	vm := setupGojaVmForHttp()
	_, err := vm.RunString(HttpScriptTemplate)
	require.NoError(t, err)

	resp := vm.Get("resp").Export().(HttpResponse)
	fmt.Printf("resp: %+v\n", resp)
}
