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
        if (txHash === undefined || txHash.length !== 2+32*2) {
            throw new Error('Invalid tx hash')
        }

        let txInfos = [];
        for (let i = 0; i < this.rpcURLs.length; i++) {
            const receiptResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genGetTxReceiptByHashReq(txHash)), 'Content-Type:application/json')
            if (receiptResp.StatusCode !== 200) {
                throw new Error('Get tx receipt error: ' + receiptResp.Status)
            }
            const receiptBody = JSON.parse(receiptResp.Body)
            const txStatus = receiptBody.result.status
            if (txStatus !== '0x1') {
                throw new Error('Tx is unsuccessful')
            }
            const blockNumberInReceiptHex = receiptBody.result.blockNumber

            const blockNumberResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genBlockNumberReq()), 'Content-Type:application/json')
            if (blockNumberResp.StatusCode !== 200) {
                throw new Error('Get block number error: ' + blockNumberResp.Status)
            }
            const blockNumberBody = JSON.parse(blockNumberResp.Body)
            const blockNumberHex = blockNumberBody.result

            const blockNumberU256 = HexToU256(blockNumberHex)
            const blockNumberInReceiptU256 = HexToU256(blockNumberInReceiptHex)
            if (blockNumberU256.Sub(blockNumberInReceiptU256).Lt(U256(this.confirmation))) {
                throw new Error('Tx is still confirming')
            }

            const headerResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genGetBlockHeaderReq(blockNumberInReceiptHex)), 'Content-Type:application/json');
            if (headerResp.StatusCode !== 200) {
                throw new Error('Get block header error: ' + headerResp.Status)
            }
            const headerBody = JSON.parse(headerResp.Body)
            const txTimestampHex = headerBody.result.timestamp

            const txResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genGetTxByHashReq(txHash)), 'Content-Type:application/json');
            if (txResp.StatusCode !== 200) {
                throw new Error('Get tx info error: ' + txResp.Status)
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

        if (!this.#checkInfos(txInfos)) {
            throw new Error('Tx/Log/Call infos are not same from different RPCs')
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

    endorseLogInfo(blockHash, sourceContract, topics) {
        if (blockHash === undefined || blockHash.length !== 2+32*2) {
            throw new Error('Invalid block hash')
        }

        if (sourceContract === undefined || sourceContract.length !== 2+20*2) {
            throw new Error('Invalid contract address')
        }

        if (topics.length > 4) {
            throw new Error('Invalid topics num')
        }

        let logInfos = []
        for (let i = 0; i < this.rpcURLs.length; i++) {
            const getLogResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genGetLogReq(blockHash, sourceContract, topics)), 'Content-Type:application/json')
            if (getLogResp.StatusCode !== 200) {
                throw new Error('Get log error: ' + getLogResp.Status)
            }
            const getLogBody = JSON.parse(getLogResp.Body)
            const getLogResults = getLogBody.result
            if (getLogResults.length === 0) {
                throw new Error('Found no logs')
            } else if (getLogResults.length > 1) {
                throw new Error('Found more than one logs')
            }

            const getLogResult = getLogResults[0]

            const blockNumberResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genBlockNumberReq()), 'Content-Type:application/json')
            if (blockNumberResp.StatusCode !== 200) {
                throw new Error('Get block number error: ' + blockNumberResp.Status)
            }
            const blockNumberBody = JSON.parse(blockNumberResp.Body)
            const blockNumberHex = blockNumberBody.result

            const blockNumberU256 = HexToU256(blockNumberHex)
            const blockNumberInLogU256 = HexToU256(getLogResult.blockNumber)
            if (blockNumberU256.Sub(blockNumberInLogU256).Lt(U256(this.confirmation))) {
                throw new Error('Log is still confirming')
            }

            const headerResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genGetBlockHeaderReq(getLogResult.blockNumber)), 'Content-Type:application/json');
            if (headerResp.StatusCode !== 200) {
                throw new Error('Get block header error: ' + headerResp.Status)
            }
            const headerBody = JSON.parse(headerResp.Body)
            const logTimestampHex = headerBody.result.timestamp

            const logInfo = {
                'chainID': this.chainID,
                'timestamp': logTimestampHex,
                'address': sourceContract,
                'topics': getLogResult.topics,
                'data': getLogResult.data,
            };
            logInfos.push(logInfo)
        }

        if (!this.#checkInfos(logInfos)) {
            throw new Error('Tx/Log/Call infos are not same from different RPCs')
        }

        const privateKey = this.bip32Key.ToPrivateKey()
        const logInfoBuf = createLogInfoBuf(logInfos[0])
        const sig = signBuf(logInfoBuf, privateKey)
        return {
            'succeeded': true,
            'message': '',
            'result': BufToB64(logInfoBuf),
            'proof': BufToB64(sig),
            'salt': '',
            'pubkey': '',
        }
    }

    endorseEthCall(sourceContract, from, data) {
        if (sourceContract === undefined || sourceContract.length !== 2+20*2) {
            throw new Error('Invalid contract address')
        }

        if (from === undefined || from.length !== 2+20*2) {
            throw new Error('Invalid from address')
        }

        if (data === undefined || data.length < 2+4*2) {
            throw new Error('Invalid data')
        }


        let callInfos = []
        for (let i = 0; i < this.rpcURLs.length; i++) {
            const blockNumberResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genBlockNumberReq()), 'Content-Type:application/json')
            if (blockNumberResp.StatusCode !== 200) {
                throw new Error('Get block number error: ' + blockNumberResp.Status)
            }
            const blockNumberBody = JSON.parse(blockNumberResp.Body)
            const blockNumberHex = blockNumberBody.result
            const blockNumberU256 = HexToU256(blockNumberHex)
            const blockNumberConfirmedU256 = blockNumberU256.Sub(U256(this.confirmation))
            const blockNumberConfirmedHex = blockNumberConfirmedU256.ToHex()

            const headerResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genGetBlockHeaderReq(blockNumberConfirmedHex)), 'Content-Type:application/json');
            if (headerResp.StatusCode !== 200) {
                throw new Error('Get block header error: ' + headerResp.Status)
            }
            const headerBody = JSON.parse(headerResp.Body)
            const timestampHex = headerBody.result.timestamp


            const ethCallResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[i], JSON.stringify(genEthCallReq(sourceContract, from, data, blockNumberConfirmedHex)), 'Content-Type:application/json');
            if (ethCallResp.StatusCode !== 200) {
                throw new Error('Get eth call error: ' + ethCallResp.Status)
            }
            const ethCallBody = JSON.parse(ethCallResp.Body)
            const out = ethCallBody.result


            const callInfo = {
                'chainID': this.chainID,
                'timestamp': timestampHex,
                'from': from,
                'to': sourceContract,
                'functionSelector': data.substring(0, 10), // 4 bytes
                'out': out,
            };
            callInfos.push(callInfo)
        }

        if (!this.#checkInfos(callInfos)) {
            throw new Error('Tx/Log/Call infos are not same from different RPCs')
        }

        const privateKey = this.bip32Key.ToPrivateKey()
        const callInfoBuf = createCallInfoBuf(callInfos[0])
        const sig = signBuf(callInfoBuf, privateKey)
        return {
            'succeeded': true,
            'message': '',
            'result': BufToB64(callInfoBuf),
            'proof': BufToB64(sig),
            'salt': '',
            'pubkey': '',
        }
    }

    #checkInfos(infos) {
        const rpcNum = this.rpcURLs.length
        if (rpcNum !== infos.length) {
            return false
        }

        if (infos.length <= 1) {
            return true;
        }

        const baseTxInfo = infos[0]
        for (let i = 1; i < infos; i++) {
            if (baseTxInfo !== infos[i]) {
                return false
            }
        }
        return true
    }
}

// ---------------------------- functions ------------------------------------

function signBuf(message, privateKey) {
    if (privateKey === undefined) {
        throw new Error('PrivateKey is undefined')
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

function createLogInfoBuf(logInfo) {
    let bb = NewBufBuilder()
    bb.Write(HexToPaddingBuf(logInfo.chainID, 32))
    bb.Write(HexToPaddingBuf(logInfo.timestamp, 32))
    bb.Write(HexToBuf(logInfo.address))
    for (let i = 0; i < logInfo.topics.length; i++) {
        bb.Write(HexToBuf(logInfo.topics[i]))
    }
    bb.Write(HexToBuf(logInfo.data))
    return bb.ToBuf()
}


function createCallInfoBuf(callInfo) {
    let bb = NewBufBuilder()
    bb.Write(HexToPaddingBuf(callInfo.chainID, 32))
    bb.Write(HexToPaddingBuf(callInfo.timestamp, 32))
    bb.Write(HexToBuf(callInfo.from))
    bb.Write(HexToBuf(callInfo.to))
    bb.Write(HexToBuf(callInfo.functionSelector))
    bb.Write(HexToBuf(callInfo.out))
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

function genGetLogReq(blockHash, sourceContract, topics) {
    return {
        'jsonrpc': JSONRPC_VERSION,
        'method': 'eth_getLogs',
        'params': [
            {
                'blockHash': blockHash,
                'address': sourceContract,
                'topics': topics
            }
        ],
        'id': JSONRPC_ID
    }
}

function genEthCallReq(sourceContract, from, data, blockNumberHex) {
    return {
        'jsonrpc': JSONRPC_VERSION,
        'method': 'eth_call',
        'params': [
            {
                'from': from,
                'to': sourceContract,
                'gasPrice': '0x1',
                'value': '0x0',
                'data': data
            },
            blockNumberHex
        ],
        'id': JSONRPC_ID
    }
}


// ---------------------------- examples ------------------------------------

function test_endorseTxResult() {
    const egvmContext = GetEGVMContext()
    const key = egvmContext.GetRootKey()
    const rpcURLs = ['https://rpc.smartbch.org', 'https://sbch-mainnet.paralinker.com/api/v1/4fd540be7cf14c437786be6415822325']
    const ccoGateway = new CcoGateway(ChainID, CONFIRMATION, rpcURLs, key)
    const endorseTxResult = ccoGateway.endorseTxInfo('0xe1b1f77471bd476a78b7fade738b3425bb8a2cef6a0c7d4fe66ce093dff61f5b')
    return JSON.stringify(endorseTxResult)
}

function test_endorseLogResult() {
    const egvmContext = GetEGVMContext()
    const key = egvmContext.GetRootKey()
    const rpcURLs = ['https://rpc.smartbch.org', 'https://sbch-mainnet.paralinker.com/api/v1/4fd540be7cf14c437786be6415822325']
    const ccoGateway = new CcoGateway(ChainID, CONFIRMATION, rpcURLs, key)
    const endorseLogResult = ccoGateway.endorseLogInfo(
        '0x95ee8003c1cdfc2c6fc67580303e5e45304575cdb6fe9b0fff0068a3550cbadc',
        '0x8bF3BAAE3aB5c6E1cA948f4F551b676E8Ab58B76',
        [
            '0x5d5cab3241b376ef7267de209f6b3c9e18abf0203218bccc442ef801f3764afc',
            '0x000000000000000000000000f78ab1ec66185a02fda96964a3a8b8c38db14703',
            '0xf78ab1ec66185a02fda96964a3a8b8c38db14703205c4503dddbaa006444f068'
        ])
    return JSON.stringify(endorseLogResult)
}

function test_endorseCallResult() {
    const egvmContext = GetEGVMContext()
    const key = egvmContext.GetRootKey()
    const rpcURLs = ['https://rpc.smartbch.org', 'https://sbch-mainnet.paralinker.com/api/v1/4fd540be7cf14c437786be6415822325']
    const ccoGateway = new CcoGateway(ChainID, CONFIRMATION, rpcURLs, key)
    const endorseEthCallResult = ccoGateway.endorseEthCall(
        '0xaEC829509AF5F8bcA30640797B38F5Ba549056FD',
        '0x3B27393cd71944Cd00Ad45273B20b6E06de95b8c',
        '0x31279d3d0000000000000000000000003b27393cd71944cd00ad45273b20b6e06de95b8c000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000b00366fbf7037e9d75e4a569ab27dab84759302')
    return JSON.stringify(endorseEthCallResult)
}


// test_endorseTxResult()
// test_endorseLogResult()
test_endorseCallResult()