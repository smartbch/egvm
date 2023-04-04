package extension

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
	HexPubkeyHash   string // for P2PKH
	HexDataElements []string // the pushed data in OP_RETURN, empty when it's P2PKH
}

func ParseTxInHex(hex string) BchTx {
	return BchTx{} //TODO
}

func SignTxAndSerialize(tx BchTx, minerFeePrice float64, privateKeys ...PrivateKey) string {
	return "" //TODO
}
