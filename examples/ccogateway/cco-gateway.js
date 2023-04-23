const JSONRPC_VERSION = '2.0';
const JSONRPC_ID = 1
const ChainID = '0x2710'
const CONFIRMATION = 10
const HTTP_METHOD_GET = 'GET'
const HTTP_METHOD_POST = 'POST'


class CcoGateway {
    constructor(chainID, confirmation, rpcURLs, bip32Key) {
        this.chainID = chainID
        this.confirmation = confirmation
        this.rpcURLs = rpcURLs
        this.bip32Key = bip32Key
    }

    endorseTxInfo(txHash) {
        // 1. send request to all rpcs to get tx info
        // 2. send request to all rpcs to get tx receipt
        // 3. check if the block confirmation condition is met
        // 4. check if the tx info from different RPCs are the same
        // 5. sign tx

        let txInfos = []
        for (let i = 0; i < this.rpcURLs.length; i++) {
            const receiptResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genGetTxReceiptByHashReq(txHash)), 'Content-Type:application/json')
            if (receiptResp.StatusCode !== 200) {
                throw new Error("Get tx receipt error: " + receiptResp.Status)
            }
            const receiptBody = JSON.parse(receiptResp.Body)
            const txStatus = receiptBody.result.status
            if (txStatus !== '0x1') {
                throw new Error("Tx is unsuccessful")
            }
            const blockNumberInReceiptHex = receiptBody.result.blockNumber

            const blockNumberResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genBlockNumberReq()), 'Content-Type:application/json')
            if (blockNumberResp.StatusCode !== 200) {
                throw new Error("Get block number error: " + blockNumberResp.Status)
            }
            const blockNumberBody = JSON.parse(blockNumberResp.Body)
            const blockNumberHex = blockNumberBody.result

            const blockNumberU256 = HexToU256(blockNumberHex)
            const blockNumberInReceiptU256 = HexToU256(blockNumberInReceiptHex)
            if (blockNumberU256.Sub(blockNumberInReceiptU256).Lt(U256(this.confirmation))) {
                throw new Error("Tx is still confirming")
            }

            const headerResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genGetBlockHeaderReq(blockNumberInReceiptHex)), 'Content-Type:application/json');
            if (headerResp.StatusCode !== 200) {
                throw new Error("Get block header error: " + headerResp.Status)
            }
            const headerBody = JSON.parse(headerResp.Body)
            const txTimestampHex = headerBody.result.timestamp

            const txResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genGetTxByHashReq(txHash)), 'Content-Type:application/json');
            if (txResp.StatusCode !== 200) {
                throw new Error("Get tx info error: " + txResp.Status)
            }

            const txRespBody = JSON.parse(txResp.Body)
            const txRespResult = txRespBody.result

            const txInfo = {
                'chainID': this.chainID,
                'timestamp': txTimestampHex,
                'txHash': txHash,
                'from': txRespResult.from,
                'to': txRespResult.to,
                'value': txRespResult.value,
                'data': txRespResult.input,
            };
            txInfos.push(txInfo)
        }

        if (!this.#checkTxInfos(txInfos)) {
            throw new Error("Tx infos are not same from different RPCs")
        }

        const privateKey = this.bip32Key.ToPrivateKey()
        const txInfoBuf = createTxInfoBuf(txInfos[0])
        const sig = signBuf(txInfoBuf, privateKey)
        return {
            'succeeded': true,
            'message': '',
            'result': BufToB64(txInfoBuf),
            'proof': BufToB64(sig),
            'salt': '',
            'pubkey': '',
        }
    }

    #checkTxInfos(txInfos) {
        const rpcNum = this.rpcURLs.length
        if (rpcNum !== txInfos.length) {
            return false
        }

        if (txInfos.length <= 1) {
            return true;
        }

        const baseTxInfo = txInfos[0]
        for (let i = 1; i < txInfos; i++) {
            if (baseTxInfo !== txInfos[i]) {
                return false
            }
        }
        return true
    }
}

function signBuf(message, privateKey) {
    if (privateKey === undefined) {
        throw new Error("PrivateKey is undefined")
    }

    const msgHash = Keccak256(message)
    const ethMsg = GetEthSignedMessage(msgHash)
    const ethMsgHash = Keccak256(ethMsg)
    let sig = privateKey.Sign(ethMsgHash)
    let sigView = new Uint8Array(sig)
    if (sigView.length === 65) {
        sigView[64] += 27
    }
    return sig
}


function createTxInfoBuf(txInfo) {
    let bb = NewBufBuilder()
    bb.Write(HexToPaddingBuf(txInfo.chainID, 32))
    bb.Write(HexToPaddingBuf(txInfo.timestamp, 32))
    bb.Write(HexToPaddingBuf(txInfo.txHash, 32))
    bb.Write(HexToBuf(txInfo.from))
    bb.Write(HexToBuf(txInfo.to))
    bb.Write(HexToPaddingBuf(txInfo.value, 32))
    bb.Write(HexToBuf(txInfo.data))
    return bb.ToBuf()
}

function genGetTxByHashReq(txHash) {
    return {
        'jsonrpc': JSONRPC_VERSION,
        'method': 'eth_getTransactionByHash',
        'params': [
            txHash
        ],
        'id': JSONRPC_ID
    }
}

function genGetTxReceiptByHashReq(txHash) {
    return {
        'jsonrpc': JSONRPC_VERSION,
        'method': 'eth_getTransactionReceipt',
        'params': [
            txHash
        ],
        'id': JSONRPC_ID
    }
}

function genBlockNumberReq() {
    return {
        'jsonrpc': JSONRPC_VERSION,
        'method': 'eth_blockNumber',
        'params': [],
        'id': JSONRPC_ID
    }
}

function genGetBlockHeaderReq(blockNumberHex) {
    return {
        'jsonrpc': JSONRPC_VERSION,
        'method': 'eth_getBlockByNumber',
        'params': [
            blockNumberHex,
            false
        ],
        'id': JSONRPC_ID
    }
}

function buf2Hex(buffer) { // buffer is an ArrayBuffer
    return [...new Uint8Array(buffer)]
        .map(x => x.toString(16).padStart(2, '0'))
        .join('')
}

function buf2Base64(buffer) {
    return btoa(String.fromCharCode.apply(null, new Uint8Array(buffer)))
}


function main() {
    const egvmContext = GetEGVMContext()
    const key = egvmContext.GetRootKey()
    const rpcURLs = ['https://rpc.smartbch.org', 'https://sbch-mainnet.paralinker.com/api/v1/4fd540be7cf14c437786be6415822325']
    const ccoGateway = new CcoGateway(ChainID, CONFIRMATION, rpcURLs, key)
    const endorseTxResult = ccoGateway.endorseTxInfo('0xe1b1f77471bd476a78b7fade738b3425bb8a2cef6a0c7d4fe66ce093dff61f5b')
    return JSON.stringify(endorseTxResult)
}

main()