package extension

import (
	"bytes"
	"encoding/hex"

	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
)

type BchTx struct {
	HexTxID  string
	Version  int32
	TxIn     []TxIn
	TxOut    []TxOut
	LockTime uint32
}

type TxIn struct {
	PreviousOutPoint OutPoint
	HexPubkey        string
	Sequence         uint32
}

type OutPoint struct {
	HexTxID string
	Index   uint32
}

type TxOut struct {
	Value           int64
	HexPubkeyHash   string   // for P2PKH
	HexDataElements []string // the pushed data in OP_RETURN, empty when it's P2PKH
}

func ParseTxInHex(hexStr string) BchTx {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		panic(err) // TODO: return error
	}

	msgTx := &wire.MsgTx{}
	err = msgTx.Deserialize(bytes.NewReader(data))
	if err != nil {
		panic(err) // TODO: return error
	}

	txIn := make([]TxIn, len(msgTx.TxIn))
	for i, msgTxIn := range msgTx.TxIn {
		txIn[i] = TxIn{
			PreviousOutPoint: OutPoint{
				HexTxID: msgTxIn.PreviousOutPoint.Hash.String(),
				Index:   msgTxIn.PreviousOutPoint.Index,
			},
			//HexPubkey: ,
			Sequence: msgTxIn.Sequence,
		}
	}

	txOut := make([]TxOut, len(msgTx.TxOut))
	for i, msgTxOut := range msgTx.TxOut {
		txOut[i] = TxOut{
			Value:           msgTxOut.Value,
			HexPubkeyHash:   getPubkeyHashHex(msgTxOut.PkScript),
			HexDataElements: getOpRetData(msgTxOut.PkScript),
		}
	}

	return BchTx{
		HexTxID:  msgTx.TxHash().String(),
		Version:  msgTx.Version,
		TxIn:     txIn,
		TxOut:    txOut,
		LockTime: msgTx.LockTime,
	}
}

// OP_DUP OP_HASH160 <pkh> OP_EQUALVERIFY OP_CHECKSIG
func getPubkeyHashHex(pkScript []byte) string {
	if len(pkScript) == 25 &&
		pkScript[0] == txscript.OP_DUP &&
		pkScript[1] == txscript.OP_HASH160 &&
		pkScript[23] == txscript.OP_EQUALVERIFY &&
		pkScript[24] == txscript.OP_CHECKSIG {

		return hex.EncodeToString(pkScript[3:23])
	}
	return ""
}

func getOpRetData(pkScript []byte) (hexData []string) {
	if len(pkScript) > 1 && pkScript[0] == txscript.OP_RETURN {
		data, _ := txscript.PushedData(pkScript)
		hexData = make([]string, len(data))
		for i, item := range data {
			hexData[i] = hex.EncodeToString(item)
		}
	}
	return
}

func SignTxAndSerialize(tx BchTx, minerFeePrice float64, privateKeys ...PrivateKey) string {
	return "" //TODO
}
