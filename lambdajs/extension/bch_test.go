package extension

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPubkeyHashHex(t *testing.T) {
	pkScript, _ := hex.DecodeString("76a914307f40d73e01af33364901d82d5614e370f905d388ac")
	pbkHash := getPubkeyHashFromPkScript(pkScript)
	require.Equal(t, "307f40d73e01af33364901d82d5614e370f905d3", pbkHash)
}

func TestGetOpRetData(t *testing.T) {
	pkScript, _ := hex.DecodeString("6a04454754581456eb561cb6f98a985f80464fa99267a462c91bdb14e94358e473941de2d75d19fa330d607e05ffab4214efc507fb38cbcae3b32d1777e54593bc07eca5a1204ea5c508a6566e76240543f8feb06fd457777be300005af3107a400000000001")
	retData := getOpRetData(pkScript)
	require.Len(t, retData, 5)
	require.Equal(t, retData[0], "45475458")
	require.Equal(t, retData[1], "56eb561cb6f98a985f80464fa99267a462c91bdb")
	require.Equal(t, retData[2], "e94358e473941de2d75d19fa330d607e05ffab42")
	require.Equal(t, retData[3], "efc507fb38cbcae3b32d1777e54593bc07eca5a1")
	require.Equal(t, retData[4], "4ea5c508a6566e76240543f8feb06fd457777be300005af3107a400000000001")
}
