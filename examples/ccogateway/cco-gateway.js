const JSONRPC_VERSION = '2.0';
const JSONRPC_ID = 1
const CHAIN_ID = '0x2710'
const CONFIRMATION = 10
const HTTP_METHOD_GET = 'GET'
const HTTP_METHOD_POST = 'POST'

const Mode = {
    proof: "proof",
    falsification: "falsification",
}

class CcoGateway {
    constructor(chainID, confirmation, rpcURLs, bip32Key, certsHash) {
        if (rpcURLs.length < 3) {
            throw new Error('No enough RPC URLs provided')
        }

        this.chainID = chainID
        this.confirmation = confirmation
        this.rpcURLs = rpcURLs
        this.bip32Key = bip32Key
        this.certsHash = certsHash
    }

    endorseTxInfo(txHash, mode) {
        if (txHash === undefined || txHash.length !== 2+32*2) {
            throw new Error('Invalid tx hash')
        }

        if (mode !== Mode.proof && mode !== Mode.falsification) {
            throw new Error('Invalid mode')
        }

        let txInfos = [];
        for (let i = 0; i < this.rpcURLs.length; i++) {
            const txInfo = this.#getTxInfo(txHash, this.rpcURLs[i])
            txInfos.push(txInfo);
        }

        const privateKey = this.bip32Key.ToPrivateKey()
        const publicKey = privateKey.GetPublicKey()

        let result = {
            'succeeded': true,
            'message': null,
            'result': null,
            'sig': null,
            'pubkey': publicKey.Hex(true),
            'missing': null, // only makes sense in proof mode
            'different': null, // only makes sense in falsification mode
            'mode': mode,
        }

        if (mode === Mode.proof) {
            const [ok, missingIndex] = this.#checkInfosInProofMode(txInfos)
            if (!ok) {
                result['succeeded'] = false
                result['message'] = 'verification failure in proof mode'
                return result
            }

            if (missingIndex > -1) {
                result['missing'] = this.rpcURLs[missingIndex]
            }

            let txInfo = txInfos[0]
            if (missingIndex === 0) {
                txInfo = txInfos[1]
            }
            const txInfoBuf = createTxInfoBufInProofMode(txInfo, this.certsHash);
            const sig = signBuf(txInfoBuf, privateKey)

            result['sig'] = BufToB64(sig)
            result['result'] = BufToB64(txInfoBuf)
            return result

        } else {
            // falsification mode
            const [ok, differentIndex] = this.#checkInfosInFalsificationMode(txInfos)
            if (!ok) {
                result['succeeded'] = false
                result['message'] = 'verification failure in falsification mode'
                return result
            }

            const txInfoBuf = createTxInfoBufInFalsificationMode(this.chainID, txHash, this.certsHash);
            const sig = signBuf(txInfoBuf, privateKey)

            if (differentIndex > -1) {
                result['different'] = this.rpcURLs[differentIndex]
            }

            result['sig'] = BufToB64(sig)
            result['result'] = BufToB64(txInfoBuf)
            return result
        }
    }

    // returns txInfo
    // Note:
    // 1. For proof mode, can allow RPC to be temporarily unavailable.
    // 2. For falsification mode, can allow query result to be null
    #getTxInfo(txHash, rpcURL) {
        const receiptResp = HttpsRequest(HTTP_METHOD_POST, rpcURL, JSON.stringify(genGetTxReceiptByHashReq(txHash)), 'Content-Type:application/json')
        if (receiptResp.StatusCode !== 200) {
            return new Error('Get tx receipt error: ' + receiptResp.Status)
        }
        const receiptBody = JSON.parse(receiptResp.Body)
        const receiptResult = receiptBody.result
        if (receiptResult === null) {
            return null
        }

        const txStatus = receiptResult.status
        if (txStatus !== '0x1') {
            return new Error('Tx is unsuccessful')
        }
        const blockNumberInReceiptHex = receiptBody.result.blockNumber

        const latestBlockNumResp = HttpsRequest(HTTP_METHOD_POST, rpcURL, JSON.stringify(genBlockNumberReq()), 'Content-Type:application/json')
        if (latestBlockNumResp.StatusCode !== 200) {
            return new Error('Get block number error: ' + latestBlockNumResp.Status)
        }
        const latestBlockNumBody = JSON.parse(latestBlockNumResp.Body)
        const latestBlockNumberHex = latestBlockNumBody.result

        const latestBlockNumU256 = HexToU256(latestBlockNumberHex)
        const blockNumberInReceiptU256 = HexToU256(blockNumberInReceiptHex)
        if (latestBlockNumU256.Sub(blockNumberInReceiptU256).Lt(U256(this.confirmation))) {
            return new Error('Tx is still confirming')
        }

        const headerResp = HttpsRequest(HTTP_METHOD_POST, rpcURL, JSON.stringify(genGetBlockHeaderReq(blockNumberInReceiptHex)), 'Content-Type:application/json');
        if (headerResp.StatusCode !== 200) {
            return new Error('Get block header error: ' + headerResp.Status)
        }
        const headerBody = JSON.parse(headerResp.Body)
        const timestamp = headerBody.result.timestamp

        const txResp = HttpsRequest(HTTP_METHOD_POST, rpcURL, JSON.stringify(genGetTxByHashReq(txHash)), 'Content-Type:application/json');
        if (txResp.StatusCode !== 200) {
            return new Error('Get tx by hash error: ' + receiptResp.Status)
        }

        const txRespBody = JSON.parse(txResp.Body)
        const txRespResult = txRespBody.result
        if (txRespResult === null) {
            return null
        }

        return {
            'chainID': this.chainID,
            'timestamp': timestamp,
            'txHash': txHash,
            'from': txRespResult.from,
            'to': txRespResult.to,
            'value': txRespResult.value,
            'data': txRespResult.input,
        }
    }

    endorseLogInfo(blockHash, sourceContract, topics, mode) {
        if (blockHash === undefined || blockHash.length !== 2+32*2) {
            throw new Error('Invalid block hash')
        }

        if (sourceContract === undefined || sourceContract.length !== 2+20*2) {
            throw new Error('Invalid contract address')
        }

        if (topics.length > 4) {
            throw new Error('Invalid topics num')
        }

        if (mode !== Mode.proof && mode !== Mode.falsification) {
            throw new Error('Invalid mode')
        }

        let logInfos = []
        for (let i = 0; i < this.rpcURLs.length; i++) {
            const logInfo = this.#getLogInfo(blockHash, sourceContract, topics, this.rpcURLs[i])
            logInfos.push(logInfo)
        }

        const privateKey = this.bip32Key.ToPrivateKey()
        const publicKey = privateKey.GetPublicKey()

        let result = {
            'succeeded': true,
            'message': null,
            'result': null,
            'sig': null,
            'pubkey': publicKey.Hex(true),
            'missing': null, // only makes sense in proof mode
            'different': null, // only makes sense in falsification mode
            'mode': mode,
        }

        if (mode === Mode.proof) {
            const [ok, missingIndex] = this.#checkInfosInProofMode(logInfos)
            if (!ok) {
                result['succeeded'] = false
                result['message'] = 'verification failure in proof mode'
                return result
            }

            if (missingIndex > -1) {
                result['missing'] = this.rpcURLs[missingIndex]
            }

            let logInfo = logInfos[0]
            if (missingIndex === 0) {
                logInfo = logInfos[1]
            }
            const logInfoBuf = createLogInfoBufInProofMode(logInfo, this.certsHash);
            const sig = signBuf(logInfoBuf, privateKey)

            result['sig'] = BufToB64(sig)
            result['result'] = BufToB64(logInfoBuf)
            return result

        } else {
            // falsification mode
            const [ok, differentIndex] = this.#checkInfosInFalsificationMode(logInfos)
            if (!ok) {
                result['succeeded'] = false
                result['message'] = 'verification failure in falsification mode'
                return result
            }

            const logInfoBuf = createLogInfoBufInFalsificationMode(this.chainID, sourceContract, topics, this.certsHash);
            const sig = signBuf(logInfoBuf, privateKey)

            if (differentIndex > -1) {
                result['different'] = this.rpcURLs[differentIndex]
            }

            result['sig'] = BufToB64(sig)
            result['result'] = BufToB64(logInfoBuf)
            return result
        }
    }

    // returns logInfo
    // Note:
    // 1. For proof mode, can allow RPC to be temporarily unavailable.
    // 2. For falsification mode, can allow query result to be null
    #getLogInfo(blockHash, sourceContract, topics, rpcURL) {
        const getLogResp = HttpsRequest(HTTP_METHOD_POST, rpcURL, JSON.stringify(genGetLogReq(blockHash, sourceContract, topics)), 'Content-Type:application/json')
        if (getLogResp.StatusCode !== 200) {
            return new Error('Get log error: ' + getLogResp.Status)
        }
        const getLogBody = JSON.parse(getLogResp.Body)
        const getLogResults = getLogBody.result
        if (getLogResults.length === 0) {
            return null
        } else if (getLogResults.length > 1) {
            return new Error('Found more than one logs')
        }

        const getLogResult = getLogResults[0]

        const latestBlockNumResp = HttpsRequest(HTTP_METHOD_POST, rpcURL, JSON.stringify(genBlockNumberReq()), 'Content-Type:application/json')
        if (latestBlockNumResp.StatusCode !== 200) {
            return new Error('Get block number error: ' + latestBlockNumResp.Status)
        }
        const latestBlockNumBody = JSON.parse(latestBlockNumResp.Body)
        const latestBlockNumHex = latestBlockNumBody.result

        const latestBlockNumU256 = HexToU256(latestBlockNumHex)
        const blockNumberInLogU256 = HexToU256(getLogResult.blockNumber)
        if (latestBlockNumU256.Sub(blockNumberInLogU256).Lt(U256(this.confirmation))) {
            return new Error('Log is still confirming')
        }

        const headerResp = HttpsRequest(HTTP_METHOD_POST, rpcURL, JSON.stringify(genGetBlockHeaderReq(getLogResult.blockNumber)), 'Content-Type:application/json');
        if (headerResp.StatusCode !== 200) {
            return new Error('Get block header error: ' + headerResp.Status)
        }
        const headerBody = JSON.parse(headerResp.Body)
        const timestamp = headerBody.result.timestamp

        return  {
            'chainID': this.chainID,
            'timestamp': timestamp,
            'address': sourceContract,
            'topics': getLogResult.topics,
            'data': getLogResult.data,
        }
    }

    endorseEthCall(sourceContract, from, data, mode) {
        if (sourceContract === undefined || sourceContract.length !== 2+20*2) {
            throw new Error('Invalid contract address')
        }

        if (from === undefined || from.length !== 2+20*2) {
            throw new Error('Invalid from address')
        }

        if (data === undefined || data.length < 2+4*2) {
            throw new Error('Invalid data')
        }

        // only supports proof mode
        if (mode !== Mode.proof) {
            throw new Error('Invalid mode')
        }

        const latestBlockNumResp = HttpsRequest(HTTP_METHOD_POST, this.rpcURLs[0], JSON.stringify(genBlockNumberReq()), 'Content-Type:application/json')
        if (latestBlockNumResp.StatusCode !== 200) {
            throw new Error('Get block number error: ' + latestBlockNumResp.Status)
        }
        const latestBlockNumBody = JSON.parse(latestBlockNumResp.Body)
        const latestBlockNumHex = latestBlockNumBody.result
        const latestBlockNumU256 = HexToU256(latestBlockNumHex)
        const blockNumberConfirmedU256 = latestBlockNumU256.Sub(U256(this.confirmation))
        const blockNumberConfirmedHex = blockNumberConfirmedU256.ToHex()

        let callInfos = []
        for (let i = 0; i < this.rpcURLs.length; i++) {
            const callInfo = this.#getEthCallInfo(sourceContract, from, data, blockNumberConfirmedHex, this.rpcURLs[i])
            callInfos.push(callInfo)
        }

        const privateKey = this.bip32Key.ToPrivateKey()
        const publicKey = privateKey.GetPublicKey()

        let result = {
            'succeeded': true,
            'message': null,
            'result': null,
            'sig': null,
            'pubkey': publicKey.Hex(true),
            'missing': null, // only makes sense in proof mode
            'different': null, // only makes sense in falsification mode
            'mode': mode,
        }

        const [ok, missingIndex] = this.#checkInfosInProofMode(callInfos)
        if (!ok) {
            result['succeeded'] = false
            result['message'] = 'verification failure in proof mode'
            return result
        }

        if (missingIndex > -1) {
            result['missing'] = this.rpcURLs[missingIndex]
        }

        let callInfo = callInfos[0]
        if (missingIndex === 0) {
            callInfo = callInfos[1]
        }
        const callInfoBuf = createCallInfoBufInProofMode(callInfo, this.certsHash);
        const sig = signBuf(callInfoBuf, privateKey)

        result['sig'] = BufToB64(sig)
        result['result'] = BufToB64(callInfoBuf)
        return result
    }

    #getEthCallInfo(sourceContract, from, data, blockNum, rpcURL) {
        const headerResp = HttpsRequest(HTTP_METHOD_POST, rpcURL, JSON.stringify(genGetBlockHeaderReq(blockNum)), 'Content-Type:application/json');
        if (headerResp.StatusCode !== 200) {
            return new Error('Get block header error: ' + headerResp.Status)
        }
        const headerBody = JSON.parse(headerResp.Body)
        const timestamp = headerBody.result.timestamp

        const ethCallResp = HttpsRequest(HTTP_METHOD_POST, rpcURL, JSON.stringify(genEthCallReq(sourceContract, from, data, blockNum)), 'Content-Type:application/json');
        if (ethCallResp.StatusCode !== 200) {
            return new Error('Get eth call error: ' + ethCallResp.Status)
        }
        const ethCallBody = JSON.parse(ethCallResp.Body)
        const out = ethCallBody.result

        return {
            'chainID': this.chainID,
            'timestamp': timestamp,
            'from': from,
            'to': sourceContract,
            'functionSelector': data.substring(0, 10), // 4 bytes
            'out': out,
        }
    }

    // return [ok, missingIndex]
    // At most one rpc is allowed to be unavailable, the others must be consistent
    // Note: missingIndex only makes sense when ok is true, -1 means no empty info
    #checkInfosInProofMode(infos) {
        if (this.rpcURLs.length !== infos.length) {
            return [false, 0]
        }

        if (haveNull(infos)) {
            return [false, 0]
        }

        let unavailableNum = 0;
        let missingIndex = -1
        let baseInfo
        for (let i = 0; i < infos.length; i++) {
            if (infos[i] instanceof Error) {
                unavailableNum++
                missingIndex = i
                continue
            }

            if (baseInfo === undefined && !(infos[i] instanceof Error)) {
                baseInfo = infos[i]
                continue
            }

            if (!isEqualInfo(baseInfo, infos[i])) {
                return [false, 0]
            }
        }

        if (unavailableNum > 1) {
            return [false, 0]
        }
        return [true, missingIndex]
    }


    // return [ok, differentIndex]
    // Return true when the result does not exist(all null), or when the query result is inconsistent with other RPCs
    // Note: only supports TxInfo/LogInfo
    // Note: differentIndex only makes sense when ok is true, -1 means the result does not exist on chain
    #checkInfosInFalsificationMode(infos) {
        if (this.rpcURLs.length !== infos.length) {
            return [false, 0]
        }

        if (haveError(infos)) {
            return [false, 0]
        }


        let differentIndex = -1;
        if (allNull(infos)) {
            return [true, differentIndex]
        }

        let baseInfo = infos[0]
        for (let i = 1; i < infos.length; i++) {
            if (!isEqualInfo(baseInfo, infos[i])) {
                // found
                differentIndex = i
                break
            }
        }

        if (differentIndex === 1) {
            if (infos[1] === infos[2]) {
                differentIndex = 0
            }
        }

        return [(differentIndex > -1), differentIndex]
    }
}

// ---------------------------- functions ------------------------------------

function isIn(v, vList) {
    for (let i = 0; i < vList.length; i++) {
        if (v === vList[i]) {
            return true
        }
    }
    return false
}

function haveError(vList) {
    for (let i = 0; i < vList.length; i++) {
        if (vList[i] instanceof Error) {
            return true
        }
    }
    return false
}

function haveNull(vList) {
    for (let i = 0; i < vList.length; i++) {
        if (vList[i] === null) {
            return true
        }
    }
    return false
}

function allNull(vList) {
    let counter = 0
    for (let i = 0; i < vList.length; i++) {
        if (vList[i] === null) {
            counter++
        }
    }
    return counter === vList.length
}



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


function createTxInfoBufInProofMode(txInfo, certsHash) {
    let bb = NewBufBuilder();
    bb.Write(HexToPaddingBuf(txInfo.chainID, 32))
    bb.Write(HexToPaddingBuf(txInfo.timestamp, 32))
    bb.Write(HexToPaddingBuf(txInfo.txHash, 32))
    bb.Write(HexToBuf(txInfo.from))
    bb.Write(HexToBuf(txInfo.to))
    bb.Write(HexToPaddingBuf(txInfo.value, 32))
    bb.Write(HexToBuf(txInfo.data))
    bb.Write(HexToPaddingBuf(certsHash, 32))
    return bb.ToBuf()
}

function createTxInfoBufInFalsificationMode(chainID, txHash, certsHash) {
    let bb = NewBufBuilder();
    bb.Write(HexToPaddingBuf(chainID, 32))
    bb.Write(HexToPaddingBuf(txHash, 32))
    bb.Write(HexToPaddingBuf(certsHash, 32))
    return bb.ToBuf()
}

function createLogInfoBufInProofMode(logInfo, certsHash) {
    let bb = NewBufBuilder()
    bb.Write(HexToPaddingBuf(logInfo.chainID, 32))
    bb.Write(HexToPaddingBuf(logInfo.timestamp, 32))
    bb.Write(HexToBuf(logInfo.address))
    for (let i = 0; i < logInfo.topics.length; i++) {
        bb.Write(HexToBuf(logInfo.topics[i]))
    }
    bb.Write(HexToBuf(logInfo.data))
    bb.Write(HexToPaddingBuf(certsHash, 32))
    return bb.ToBuf()
}

function createLogInfoBufInFalsificationMode(chainID, contractAddress, topics, certsHash) {
    let bb = NewBufBuilder()
    bb.Write(HexToPaddingBuf(chainID, 32))
    bb.Write(HexToBuf(contractAddress))
    for (let i = 0; i < topics.length; i++) {
        bb.Write(topics[i])
    }
    bb.Write(HexToPaddingBuf(certsHash, 32))
    return bb.ToBuf()
}


function createCallInfoBufInProofMode(callInfo, certsHash) {
    let bb = NewBufBuilder()
    bb.Write(HexToPaddingBuf(callInfo.chainID, 32))
    bb.Write(HexToPaddingBuf(callInfo.timestamp, 32))
    bb.Write(HexToBuf(callInfo.from))
    bb.Write(HexToBuf(callInfo.to))
    bb.Write(HexToBuf(callInfo.functionSelector))
    bb.Write(HexToBuf(callInfo.out))
    bb.Write(HexToPaddingBuf(certsHash, 32))
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

function isEqualInfo(a, b) {
    return JSON.stringify(a) === JSON.stringify(b)
}


// ---------------------------- examples ------------------------------------


const testRPCURLs = ['https://rpc.smartbch.org', 'https://sbch-mainnet.paralinker.com/api/v1/4fd540be7cf14c437786be6415822325', 'https://smartbch.greyh.at']
// const testRPCURLsWithOneUnavailableRPC = ['https://rpc.smartbch.org', 'https://sbch-mainnet.paralinker.com/api/v2/4fd540be7cf14c437786be6415822325', 'https://smartbch.greyh.at']

function test_endorseTxResultInProofMode() {
    const egvmContext = GetEGVMContext()
    const certsHash = egvmContext.GetCertsHash()
    const certsHashHex = '0x' + BufToHex(certsHash)
    const key = egvmContext.GetRootKey()
    const ccoGateway = new CcoGateway(CHAIN_ID, CONFIRMATION, testRPCURLs, key,  certsHashHex)
    const endorseTxResult = ccoGateway.endorseTxInfo(
        '0xe1b1f77471bd476a78b7fade738b3425bb8a2cef6a0c7d4fe66ce093dff61f5b', Mode.proof)
    return JSON.stringify(endorseTxResult)
}

function test_endorseTxResultInFalsificationMode() {
    const egvmContext = GetEGVMContext()
    const certsHash = egvmContext.GetCertsHash()
    const certsHashHex = '0x' + BufToHex(certsHash)
    const key = egvmContext.GetRootKey()
    const ccoGateway = new CcoGateway(CHAIN_ID, CONFIRMATION, testRPCURLs, key,  certsHashHex)
    const endorseTxResult = ccoGateway.endorseTxInfo(
        '0xf1b1f77471bd476a78b7fade738b3425bb8a2cef6a0c7d4fe66ce093dff61f5b', Mode.falsification)
    return JSON.stringify(endorseTxResult)
}

function test_endorseLogResultInProofMode() {
    const egvmContext = GetEGVMContext()
    const certsHash = egvmContext.GetCertsHash()
    const certsHashHex = '0x' + BufToHex(certsHash)
    const key = egvmContext.GetRootKey()
    const ccoGateway = new CcoGateway(CHAIN_ID, CONFIRMATION, testRPCURLs, key,  certsHashHex)
    const endorseLogResult = ccoGateway.endorseLogInfo(
        '0x95ee8003c1cdfc2c6fc67580303e5e45304575cdb6fe9b0fff0068a3550cbadc',
        '0x8bF3BAAE3aB5c6E1cA948f4F551b676E8Ab58B76',
        [
            '0x5d5cab3241b376ef7267de209f6b3c9e18abf0203218bccc442ef801f3764afc',
            '0x000000000000000000000000f78ab1ec66185a02fda96964a3a8b8c38db14703',
            '0xf78ab1ec66185a02fda96964a3a8b8c38db14703205c4503dddbaa006444f068'
        ], Mode.proof)
    return JSON.stringify(endorseLogResult)
}

function test_endorseLogResultInFalsificationMode() {
    const egvmContext = GetEGVMContext()
    const certsHash = egvmContext.GetCertsHash()
    const certsHashHex = '0x' + BufToHex(certsHash)
    const key = egvmContext.GetRootKey()
    const ccoGateway = new CcoGateway(CHAIN_ID, CONFIRMATION, testRPCURLs, key,  certsHashHex)
    const endorseLogResult = ccoGateway.endorseLogInfo(
        '0x95ee8003c1cdfc2c6fc67580303e5e45304575cdb6fe9b0fff0068a3550cbadc',
        '0x8bF3BAAE3aB5c6E1cA948f4F551b676E8Ab58B76',
        [
            '0x5d5cab3241b376ef7267de209f6b3c9e18abf0203218bccc442ef801f3764afc',
            '0x000000000000000000000000f78ab1ec66185a02fda96964a3a8b8c38db14703',
            '0xf78ab1ec66185a02fda96964a3a8b8c38db14703205c4503dddbaa006444f068'
        ], Mode.falsification)
    return JSON.stringify(endorseLogResult)
}

function test_endorseCallResultInProofMode() {
    const egvmContext = GetEGVMContext()
    const certsHash = egvmContext.GetCertsHash()
    const certsHashHex = '0x' + BufToHex(certsHash)
    const key = egvmContext.GetRootKey()
    const ccoGateway = new CcoGateway(CHAIN_ID, CONFIRMATION, testRPCURLs, key, certsHashHex)
    const endorseEthCallResult = ccoGateway.endorseEthCall(
        '0x22a9D210ba154994ad1477F585602eC41b99b931',
        '0x32c7d35F6Ac14437318035E4AECB3e9e8a84D556',
        '0x2542f3d500000000000000000000000022a9d210ba154994ad1477f585602ec41b99b9315fe7f977e71dba2ea1a68e21057beebb9be2ac30c6410aa38d4f3fbe41dcffd222a9d210ba154994ad1477f585602ec41b99b931000038400063b0082000010f00000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000000016d1e088b8d15320b2e0b4fe4adc74b7401789a76d3c55aa2046d1133e8e5eacb530af0c450dc5292b5c7b06a84e26cc5244d8d42e5aa925877638c36f2a81c4b00000000000000000000000000000000000000000000000000000000000027100000000000000000000000000000000000000000000000000000000063ae473600000000000000000000000032c7d35f6ac14437318035e4aecb3e9e8a84d55600000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000000033e38a7272680c978d8255418e1729d3092ba064f5a3880c96e4827f94111bc8f22a9d210ba154994ad1477f585602ec41b99b931000038400063b0082000010f00000000000000000000000022a9d210ba154994ad1477f585602ec41b99b93100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000063ae4736'
        , Mode.proof)
    return JSON.stringify(endorseEthCallResult)
}


// test_endorseTxResultInProofMode()
// test_endorseTxResultInFalsificationMode()
// test_endorseLogResultInProofMode()
// test_endorseLogResultInFalsificationMode()
test_endorseCallResultInProofMode()
