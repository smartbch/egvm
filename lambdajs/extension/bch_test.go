package extension

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	ecies "github.com/ecies/go/v2"
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
	require.Equal(t, "45475458", retData[0])
	require.Equal(t, "56eb561cb6f98a985f80464fa99267a462c91bdb", retData[1])
	require.Equal(t, "e94358e473941de2d75d19fa330d607e05ffab42", retData[2])
	require.Equal(t, "efc507fb38cbcae3b32d1777e54593bc07eca5a1", retData[3])
	require.Equal(t, "4ea5c508a6566e76240543f8feb06fd457777be300005af3107a400000000001", retData[4])
}

func TestParseTxInHex(t *testing.T) {
	// https://blockchair.com/bitcoin-cash/transaction/c1d33d4c03c81f8dee2f176b944bc05eb08524163698d4d397a8fb9ca7cd2651
	txHex := "020000000147c8c5a1f4d7d5e3846a7e945daf634722340f617be0ff8736e668d7ee7d9fb402000000644128fd33544f9530b1a8ae03340bdfe9385324adf0ecefc39d53e6fddf9afdb64ccf3ef12bc692045d87e90380919429d3c5bafc29b51515aa0a992dd0d085663e4121031c60b05831b6f3c31739856575cde27d97d9fe926a63d51abce4a0c16b4108be00000000030000000000000000666a04454754581456eb561cb6f98a985f80464fa99267a462c91bdb14e94358e473941de2d75d19fa330d607e05ffab4214efc507fb38cbcae3b32d1777e54593bc07eca5a1204ea5c508a6566e76240543f8feb06fd457777be300005af3107a40000000000110270000000000001976a9148097f6fbaa0dfdfe4f064bb650324c5e8018242088acca331e00000000001976a914307f40d73e01af33364901d82d5614e370f905d388ac00000000"

	tx, err := ParseTxInHex(txHex)
	require.NoError(t, err)
	require.Equal(t, int32(2), tx.Version)
	require.Equal(t, uint32(0), tx.LockTime)
	require.Equal(t, "c1d33d4c03c81f8dee2f176b944bc05eb08524163698d4d397a8fb9ca7cd2651", tx.HexTxID)
	require.Len(t, tx.TxIn, 1)
	require.Len(t, tx.TxOut, 3)
	require.Equal(t, TxIn{
		PreviousOutPoint: OutPoint{
			HexTxID: "b49f7deed768e63687ffe07b610f34224763af5d947e6a84e3d5d7f4a1c5c847",
			Index:   2,
		},
		HexPubkey: "031c60b05831b6f3c31739856575cde27d97d9fe926a63d51abce4a0c16b4108be",
		Sequence:  0,
	}, tx.TxIn[0])
	require.Equal(t, TxOut{
		Value:         0,
		HexPubkeyHash: "",
		HexDataElements: []string{
			"45475458",
			"56eb561cb6f98a985f80464fa99267a462c91bdb",
			"e94358e473941de2d75d19fa330d607e05ffab42",
			"efc507fb38cbcae3b32d1777e54593bc07eca5a1",
			"4ea5c508a6566e76240543f8feb06fd457777be300005af3107a400000000001",
		},
	}, tx.TxOut[0])
	require.Equal(t, TxOut{
		Value:           10000,
		HexPubkeyHash:   "8097f6fbaa0dfdfe4f064bb650324c5e80182420",
		HexDataElements: nil,
	}, tx.TxOut[1])
	require.Equal(t, TxOut{
		Value:           1979338,
		HexPubkeyHash:   "307f40d73e01af33364901d82d5614e370f905d3",
		HexDataElements: nil,
	}, tx.TxOut[2])
}

func TestSignTxAndSerialize(t *testing.T) {
	txHex := "020000000147c8c5a1f4d7d5e3846a7e945daf634722340f617be0ff8736e668d7ee7d9fb402000000644128fd33544f9530b1a8ae03340bdfe9385324adf0ecefc39d53e6fddf9afdb64ccf3ef12bc692045d87e90380919429d3c5bafc29b51515aa0a992dd0d085663e4121031c60b05831b6f3c31739856575cde27d97d9fe926a63d51abce4a0c16b4108be00000000030000000000000000666a04454754581456eb561cb6f98a985f80464fa99267a462c91bdb14e94358e473941de2d75d19fa330d607e05ffab4214efc507fb38cbcae3b32d1777e54593bc07eca5a1204ea5c508a6566e76240543f8feb06fd457777be300005af3107a40000000000110270000000000001976a9148097f6fbaa0dfdfe4f064bb650324c5e8018242088acca331e00000000001976a914307f40d73e01af33364901d82d5614e370f905d388ac00000000"
	tx, err := ParseTxInHex(txHex)
	require.NoError(t, err)

	privKey, err := ecies.GenerateKey()
	require.NoError(t, err)

	signedTxHex, err := SignTxAndSerialize(*tx, PrivateKey{key: privKey})
	require.NoError(t, err)
	fmt.Println(signedTxHex)

	signedTx, err := ParseTxInHex(signedTxHex)
	require.NoError(t, err)
	require.Equal(t, int32(2), signedTx.Version)
	require.Equal(t, uint32(0), signedTx.LockTime)
	require.Len(t, signedTx.TxIn, 1)
	require.Len(t, signedTx.TxOut, 3)
	require.Len(t, signedTx.TxOut[0].HexDataElements, 5)
	require.Equal(t, "8097f6fbaa0dfdfe4f064bb650324c5e80182420", signedTx.TxOut[1].HexPubkeyHash)
}
