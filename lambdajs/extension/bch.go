package extension

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
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
	Value            int64
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

func ParseTxInHex(hexStr string) (*BchTx, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string: %w", err)
	}

	msgTx := &wire.MsgTx{}
	err = msgTx.Deserialize(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize tx: %w", err)
	}

	txIn := make([]TxIn, len(msgTx.TxIn))
	for i, msgTxIn := range msgTx.TxIn {
		pubkey, err := getPubkeyFromSigScript(msgTxIn.SignatureScript)
		if err != nil {
			return nil, fmt.Errorf("failed to get pubkey from unlocking script of input#%d, %w", i, err)
		}

		txIn[i] = TxIn{
			PreviousOutPoint: OutPoint{
				HexTxID: msgTxIn.PreviousOutPoint.Hash.String(),
				Index:   msgTxIn.PreviousOutPoint.Index,
			},
			HexPubkey: pubkey,
			Sequence:  msgTxIn.Sequence,
		}
	}

	txOut := make([]TxOut, len(msgTx.TxOut))
	for i, msgTxOut := range msgTx.TxOut {
		txOut[i] = TxOut{
			Value:           msgTxOut.Value,
			HexPubkeyHash:   getPubkeyHashFromPkScript(msgTxOut.PkScript),
			HexDataElements: getOpRetData(msgTxOut.PkScript),
		}
	}

	return &BchTx{
		HexTxID:  msgTx.TxHash().String(),
		Version:  msgTx.Version,
		TxIn:     txIn,
		TxOut:    txOut,
		LockTime: msgTx.LockTime,
	}, nil
}

// <sig> <pubkey>
func getPubkeyFromSigScript(sigScript []byte) (string, error) {
	pushes, err := txscript.PushedData(sigScript)
	if err != nil {
		return "", fmt.Errorf("failed to parse sigScript: %w", err)
	}
	if len(pushes) != 2 {
		return "", fmt.Errorf("invalid sigScript")
	}
	//if len(pushes[1]) != 65 {
	//	return "", fmt.Errorf("invalid pubkey")
	//}
	return hex.EncodeToString(pushes[1]), nil
}

// OP_DUP OP_HASH160 <pkh> OP_EQUALVERIFY OP_CHECKSIG
func getPubkeyHashFromPkScript(pkScript []byte) string {
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

func SignTxAndSerialize(tx BchTx, privateKeys ...PrivateKey) (string, error) {
	if len(tx.TxIn) != len(privateKeys) {
		return "", fmt.Errorf("length of tx inputs and private keys mismatch")
	}

	msgTx := wire.NewMsgTx(tx.Version)
	msgTx.LockTime = tx.LockTime
	for idx, in := range tx.TxIn {
		prevTxHash, err := chainhash.NewHashFromStr(in.PreviousOutPoint.HexTxID)
		if err != nil {
			return "", fmt.Errorf("failed to decode prevTxHash of input#%d: %w", idx, err)
		}
		msgTx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: wire.OutPoint{
				Hash:  *prevTxHash,
				Index: in.PreviousOutPoint.Index,
			},
		})
	}
	for idx, out := range tx.TxOut {
		pubkeyHash, err := hex.DecodeString(out.HexPubkeyHash)
		if err != nil {
			return "", fmt.Errorf("failed to decode pubkeyHash of output#%d, %w", idx, err)
		}
		pkScript, err := payToPubKeyHashScript(pubkeyHash)
		if err != nil {
			return "", fmt.Errorf("failed to create pkScript of output#%d, %w", idx, err)
		}
		msgTx.AddTxOut(&wire.TxOut{
			Value:    out.Value,
			PkScript: pkScript,
		})
	}

	sigHashes := txscript.NewTxSigHashes(msgTx)
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	for idx, in := range msgTx.TxIn {
		privKey := (*bchec.PrivateKey)(privateKeys[idx].toECDSA())
		pubKeyBytes := privKey.PubKey().SerializeCompressed()
		pubkeyHash := bchutil.Hash160(pubKeyBytes)

		pubkeyScript, err := payToPubKeyHashScript(pubkeyHash)
		if err != nil {
			return "", fmt.Errorf("failed to create locaking script of input#%d, %w", idx, err)
		}

		sigHash, err := txscript.CalcSignatureHash(pubkeyScript, sigHashes, hashType, msgTx,
			idx, tx.TxIn[idx].Value, true)
		if err != nil {
			return "", fmt.Errorf("failed to calc signature hash of input#%d, %w", idx, err)
		}

		sig, err := signTxSigHashECDSA(privKey, sigHash, hashType)
		if err != nil {
			return "", fmt.Errorf("failed to sign sigHash of input#%d, %w", idx, err)
		}

		sigScript, err := txscript.NewScriptBuilder().AddData(sig).AddData(pubKeyBytes).Script()
		if err != nil {
			return "", fmt.Errorf("failed to create unlocking script of input#%d, %w", idx, err)
		}
		in.SignatureScript = sigScript
	}

	return hex.EncodeToString(msgTxToBytes(msgTx)), nil
}

func payToPubKeyHashScript(pubKeyHash []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_DUP).
		AddOp(txscript.OP_HASH160).
		AddData(pubKeyHash).
		AddOp(txscript.OP_EQUALVERIFY).
		AddOp(txscript.OP_CHECKSIG).
		Script()
}

func signTxSigHashECDSA(privKey *bchec.PrivateKey,
	hash []byte, hashType txscript.SigHashType) ([]byte, error) {

	signature, err := privKey.SignECDSA(hash)
	if err != nil {
		return nil, fmt.Errorf("cannot sign tx input: %s", err)
	}

	return append(signature.Serialize(), byte(hashType)), nil
}

func msgTxToBytes(tx *wire.MsgTx) []byte {
	var buf bytes.Buffer
	_ = tx.Serialize(&buf)
	return buf.Bytes()
}
