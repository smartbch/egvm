package extension

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

const (
	HttpScriptTemplate = `
		const resp = HttpRequest('GET', 'https://elfinauth.paralinker.io/smartbch/eh_ping', '', 'Content-Type:application/json')
	`
)

func setupGojaVmForHttp() *goja.Runtime {
	vm := goja.New()
	vm.Set("HttpRequest", HttpRequest)
	return vm
}

func TestHttpRequest(t *testing.T) {
	vm := setupGojaVmForHttp()
	_, err := vm.RunString(HttpScriptTemplate)
	require.NoError(t, err)

	resp := vm.Get("resp").Export().(HttpResponse)
	require.EqualValues(t, 200, resp.StatusCode)
	require.EqualValues(t, `{"isSuccess":true,"message":"pong"}`, resp.Body)
}
