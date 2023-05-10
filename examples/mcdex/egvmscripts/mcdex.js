const JSONRPC_VERSION = '2.0'
const JSONRPC_ID = 1
const HTTP_METHOD_GET = 'GET'
const HTTP_METHOD_POST = 'POST'
const OrderSide = {
    buy: "buy",
    sell: "sell",
}

// No blocks to be synchronized before MCDEX is deployed to the mainnet
// chainId => initial height
const INITIAL_HEIGHT_MAP = {
    '0x2710': '0x91feb1',
    '0x38': '0x1ac8778',
}

// contract address => topics
const FOCUSED_EVENTS = {
    '0x77CB87b57F54667978Eb1B199b28a0db8C8E1c0B': ['0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef']
}

// ----------------------------------------------------------------

class Oracle {
    constructor() {
        this.chainMap = new Map()           // chainId => rpcURLs
        this.processedHeights = new Map()   // chainId => height; record the heights that have been processed
        this.chainIds = []                  // keep chainIds index order
    }

    // parameters: (string(hex), string array)
    setChain(chainId, rpcURLs) {
        if (rpcURLs.length < 1) { //TODO: at least 4 rpcs for each chain
            throw new Error('No enough RPC URLs provided')
        }

        // if add new chain, record the index
        if (this.chainMap.get(chainId) === undefined) {
            this.chainIds.push(chainId)
            const initialHeight = INITIAL_HEIGHT_MAP[chainId]
            if (initialHeight === undefined) {
                throw new Error(`No initial height for chainId: ${chainId}`)
            }
            this.processedHeights.set(chainId, initialHeight)
        }
        this.chainMap.set(chainId, rpcURLs)
    }

    getProcessedHeights() {
        let results = []
        for (let i = 0; i < this.chainIds.length; i++) {
            results.push(this.processedHeights.get(this.chainIds[i]))
        }
        return results
    }

    getProcessedHeightByChainId(chainId) {
        if (!isIn(chainId, this.chainIds)) {
            throw new Error(`Invalid chainId ${chainId}`)
        }
        return this.processedHeights.get(chainId)
    }

    setProcessedHeight(chainId, newHeight) {
        if (!isIn(chainId, this.chainIds)) {
            throw new Error(`Invalid chainId ${chainId}`)
        }
        this.processedHeights.set(chainId, newHeight);
    }

    reset() {
        this.chainIds.clear()
        this.chainMap.clear()
    }

    // Get latest height and latest timestamp for all chains
    // return: (string[], U256)
    getLatestHeightFromMultiChains() {
        let latestHeights = []
        let latestTimestamps = []

        for (let i = 0; i < this.chainIds.length; i++) {
            let chainId = this.chainIds[i]
            let rpcURLs = this.chainMap.get(chainId)

            let latestHeightsInChain = []
            for (let j = 0; j < rpcURLs.length; j++) {
                const latestBlockNumResp = HttpsRequest(HTTP_METHOD_POST, rpcURLs[j], JSON.stringify(genBlockNumberReq()), 'Content-Type:application/json')
                if (latestBlockNumResp.StatusCode !== 200) {
                    latestHeightsInChain.push(new Error('Get block number error: ' + latestBlockNumResp.Status))
                }

                const latestBlockNumBody = JSON.parse(latestBlockNumResp.Body)
                const latestBlockNumberHex = latestBlockNumBody.result
                latestHeightsInChain.push(latestBlockNumberHex)
            }

            let [latestHeightInChain, ok] = checkLatestHeightInProofMode(latestHeightsInChain)
            if (!ok) {
                throw new Error('Check latest height in proof mode failed')
            }

            latestHeights.push(latestHeightInChain)
            SleepMs(300)

            const headerResp = HttpsRequest(HTTP_METHOD_POST, rpcURLs[0], JSON.stringify(genGetBlockHeaderReq(latestHeightInChain)), 'Content-Type:application/json');
            if (headerResp.StatusCode !== 200) {
                throw new Error('Get block header error: ' + headerResp.Status)
            }
            const headerBody = JSON.parse(headerResp.Body)
            const timestamp = headerBody.result.timestamp
            latestTimestamps.push(timestamp)
            SleepMs(300)
        }

        return [latestHeights, findMinHex(latestTimestamps)]
    }

    // blockInfo:
    // {
    //     chainId: <string>,
    //     height: <string>,
    //     timestamp: <string>,
    //     hash: <string>,
    //     events: <event[]>
    // }

    // event:
    // {
    //     address: <string>,
    //     topics: <string[]>,
    //     data: <string>,
    //     logIndex: <string>,
    //     transactionIndex: <string>
    // }

    // Given latest height list and latest timestamp, find globally-confirmed blockInfos with events
    // parameters: string[], string
    // return: blockInfo[]
    getNewGloballyConfirmedBlocks(latestHeights, latestTimestamp) {
        if (latestHeights.length !== this.chainIds.length) {
            throw new Error(`The number of latestHeights is invalid`)
        }

        if (latestTimestamp === null || latestTimestamp === '') {
            throw new Error(`Invalid latestTimestamp`)
        }

        let globallyConfirmedBlockInfos = [];
        for (let i = 0; i < this.chainIds.length; i++) {
            const currentChainId = this.chainIds[i]
            let rpcURLs = this.chainMap.get(currentChainId)
            const defaultRpcURL = rpcURLs[0]

            const currentHeight = latestHeights[i]
            const nextHeight = HexToU256(this.processedHeights.get(currentChainId)).Incr()

            // record blocks with focused events
            // for current chainId
            // height => blockInfo
            let blockMap = new Map()
            for (const sourceContract in FOCUSED_EVENTS) {
                if (!FOCUSED_EVENTS.hasOwnProperty(sourceContract)){
                    continue
                }

                const topics = FOCUSED_EVENTS[sourceContract]
                const getLogResp = HttpsRequest(HTTP_METHOD_POST, defaultRpcURL,
                    JSON.stringify(genGetLogReq(nextHeight.ToHex(), latestHeights[i], sourceContract, topics)), 'Content-Type:application/json');
                if (getLogResp.StatusCode !== 200) {
                    throw new Error('Get log error: ' + getLogResp.Status)
                }

                const getLogBody = JSON.parse(getLogResp.Body)
                const getLogResults = getLogBody.result
                if (getLogResults.length === 0) {
                    continue
                }

                for (let j = 0; j < getLogResults.length; j++) {
                    const headerResp = HttpsRequest(HTTP_METHOD_POST, defaultRpcURL,
                        JSON.stringify(genGetBlockHeaderReq(getLogResults[j].blockNumber)), 'Content-Type:application/json');
                    if (headerResp.StatusCode !== 200) {
                        throw new Error('Get block header error: ' + headerResp.Status);
                    }

                    const headerBody = JSON.parse(headerResp.Body)
                    const timestamp = headerBody.result.timestamp
                    if (HexToU256(timestamp).Gt(HexToU256(latestTimestamp)) || HexToU256(getLogResults[j].blockNumber).Gte(HexToU256(currentHeight))) {
                        break
                    }

                    let event = {
                        address: getLogResults[j].address,
                        topics: getLogResults[j].topics,
                        data: getLogResults[j].data,
                        logIndex: getLogResults[j].logIndex,
                        transactionIndex: getLogResults[j].transactionIndex
                    }

                    let blockInfo = blockMap.get(getLogResults[j].blockNumber)
                    if (blockInfo === undefined) {
                        blockInfo = {
                            chainId: currentChainId,
                            height: getLogResults[j].blockNumber,
                            timestamp: timestamp,
                            hash: headerBody.result.hash,
                            events: [],
                        }
                    }
                    blockInfo.events.push(event)
                    blockMap.set(getLogResults[j].blockNumber, blockInfo)
                }
            }

            for (let blockInfo of blockMap.values()) {
                blockInfo.events.sort(eventComparator)
                globallyConfirmedBlockInfos.push(blockInfo)
            }
            SleepMs(300)
        }

        globallyConfirmedBlockInfos.sort(blockInfoComparator)
        return globallyConfirmedBlockInfos
    }


    // Return first unprocessed blockInfo, and update the processedHeight
    // parameter: blockInfo[]
    // return: blockInfo
    getNewBlockForProcessing(blockInfos) {
        if (blockInfos.length === 0) {
            throw new Error('Empty blockInfos')
        }
        blockInfos.sort(blockInfoComparator)

        const currentProcessedHeight = this.getProcessedHeightByChainId(blockInfos[0].chainId)
        if (HexToU256(blockInfos[0].height).Lte(HexToU256(currentProcessedHeight)) ) {
            throw new Error('The first blockInfo has been already processed')
        }

        let firstInfo = blockInfos.shift();
        this.setProcessedHeight(firstInfo.chainId, firstInfo.height)
        return firstInfo
    }

    // Given processed heights and timestamp, check them whether they are all globally-conformed until this timestamp
    // Note: the uncheckedHeights order must be same as chainIds order
    // parameters: string[], string
    // return: bool
    isAllGloballyConfirmed(uncheckedHeights, checkPointTimestamp) {
        if (uncheckedHeights.length !== this.chainIds.length) {
            throw new Error(`The number of uncheckedHeights is invalid`)
        }

        if (checkPointTimestamp === null || checkPointTimestamp === '') {
            throw new Error(`Invalid checkPointTimestamp`)
        }

        for (let i = 0; i < this.chainIds.length; i++) {
            const currentChainId = this.chainIds[i]
            let rpcURLs = this.chainMap.get(currentChainId)
            const defaultRpcURL = rpcURLs[0]

            // 1. check latest height for each chain
            // 2. check block header
            const latestBlockNumResp = HttpsRequest(HTTP_METHOD_POST, defaultRpcURL, JSON.stringify(genBlockNumberReq()), 'Content-Type:application/json')
            if (latestBlockNumResp.StatusCode !== 200) {
                throw new Error('Get block number error: ' + latestBlockNumResp.Status)
            }

            const latestBlockNumBody = JSON.parse(latestBlockNumResp.Body)
            const latestBlockNumberHex = latestBlockNumBody.result
            if (HexToU256(uncheckedHeights[i]).Gt(HexToU256(latestBlockNumberHex))) {
                return false
            }

            const headerResp = HttpsRequest(HTTP_METHOD_POST, rpcURLs[0], JSON.stringify(genGetBlockHeaderReq(latestBlockNumberHex)), 'Content-Type:application/json');
            if (headerResp.StatusCode !== 200) {
                throw new Error('Get block header error: ' + headerResp.Status)
            }
            const headerBody = JSON.parse(headerResp.Body)
            const timestamp = headerBody.result.timestamp
            if (HexToU256(checkPointTimestamp).Gt(HexToU256(timestamp))) {
                return false
            }

            SleepMs(300)
        }

        return true
    }
}


// only for events in same height
// sorted by transactionIndex and logIndex
function eventComparator(a, b) {
    if (HexToU256(a.transactionIndex).Lt(HexToU256(b.transactionIndex))) {
        return -1
    } else if (HexToU256(a.transactionIndex).Gt(HexToU256(b.transactionIndex))) {
        return 1
    } else if (HexToU256(a.logIndex).Lt(HexToU256(b.logIndex))) {
        return -1
    }
    return 1;
}


// The block with the smaller timestamp is ranked first,
// if the timestamps are the same, the hash will be compared
function blockInfoComparator(a, b) {
    if (HexToU256(a.timestamp).Lt(HexToU256(b.timestamp))) {
        return -1
    } else if (HexToU256(a.timestamp).Gt(HexToU256(b.timestamp))) {
        return 1
    } else {
        return BufCompare(HexToBuf(a.hash), HexToBuf(b.hash))
    }
}

function hex2Number(hex) {
    return parseInt(hex, 16)
}

function number2Hex(num) {
    return '0x' + num.toString(16);
}

function findMinHex(hexList) {
    let numList = []
    for (let i = 0; i < hexList.length; i++) {
        numList.push(hex2Number(hexList[i]))
    }
    return number2Hex(Math.min(...numList));
}

function findMaxHex(hexList) {
    let numList = []
    for (let i = 0; i < hexList.length; i++) {
        numList.push(hex2Number(hexList[i]))
    }
    return number2Hex(Math.max(...numList));
}

function checkLatestHeightInProofMode(latestHeightsInChain) {
    if (latestHeightsInChain.length === 0) {
        return ['0x0', false]
    }

    let unavailableNum = 0
    let nonErrorOrNullHeights = []
    for (let i = 0; i < latestHeightsInChain.length; i++) {
        if (latestHeightsInChain[i] instanceof Error) {
            unavailableNum++
            continue
        }

        if (latestHeightsInChain[i] === '' || latestHeightsInChain[i] === null) {
            return ['0x0', false]
        }

        nonErrorOrNullHeights.push(latestHeightsInChain[i])
    }

    return [findMinHex(nonErrorOrNullHeights), unavailableNum <= 1];
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

function genGetLogReq(fromBlock, toBlock, sourceContract, topics) {
    return {
        'jsonrpc': JSONRPC_VERSION,
        'method': 'eth_getLogs',
        'params': [
            {
                'fromBlock': fromBlock,
                'toBlock': toBlock,
                'address': sourceContract,
                'topics': topics
            }
        ],
        'id': JSONRPC_ID
    }
}

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


// ----------------------------------------------------------------

class Order {
    // U256, number, U256, U256, OrderSide, string(address)
    constructor(price, height, totalAmount, side, owner) {
        this.price = price;
        this.height = height;
        this.totalAmount = totalAmount;
        this.remainAmount = totalAmount;
        this.side = side;
        this.owner = owner;
    }

    getPrice() {
        return this.price
    }

    getAmount() {
        return this.remainAmount
    }

    getHeight() {
        return this.height
    }

    getSide() {
        return this.side
    }

    getOwner() {
        return this.owner
    }

    getHash() {
        return Sha256(UTF8StrToBuf(this.owner));
    }

    deal(againstOrder, amount, price) {
        Printf("Deal: %s|%v-%s|%v %v price:%v\n", this.getOwner(), this.getAmount(), againstOrder.getOwner(), againstOrder.getAmount(), amount, price)
        this.remainAmount = this.remainAmount.Sub(amount)
        againstOrder.remainAmount = againstOrder.remainAmount.Sub(amount)
        // TODO: add deal record
    }

}

// return negative number if a has more priority than b
function orderComparator(a, b) {
    if (a.getSide() === OrderSide.sell && a.getPrice().Lt(b.getPrice()) ||
        a.getSide() === OrderSide.buy && a.getPrice().Gt(b.getPrice())) {
        return -1
    } else if (a.getSide() === OrderSide.sell && a.getPrice().Gt(b.getPrice()) ||
        a.getSide() === OrderSide.buy && a.getPrice().Lt(b.getPrice())) {
        return 1
    } else if (a.getHeight() < b.getHeight()) {
        return -1
    } else if (a.getHeight() > b.getHeight()) {
        return 1;
    } else {
        return BufCompare(a.getHash(), b.getHash())
    }
}

// ----------------------------------------------------------------

// parameters: (U256, U256, U256, Order[], Order[])
// return: void
function Match(highPrice, midPrice, lowPrice, bidList, askList) {
    bidList.sort(orderComparator)
    askList.sort(orderComparator)

    while (bidList.length > 0 && askList.length > 0 && askList[0].getPrice().Lte(bidList[0].getPrice())) {
        const price = GetExecutionPrice(highPrice, midPrice, lowPrice, bidList, askList)
        ExecuteOrderList(price, bidList, askList)
    }
}

// Given price, execute the orders in bidList and askList inplace
// parameters: (U256, U256, U256, Order[], Order[])
function ExecuteOrderList(price, bidList, askList) {
	while (true) {
		if (askList.length === 0 || bidList.length === 0 ||
			bidList[0].getPrice().Lt(price) || askList[0].getPrice().Gt(price)) {
			break
		}

        ExecuteOrder(price, bidList[0], askList)
        clearZeroOrderList(bidList)

		if (askList.length === 0 || bidList.length === 0 ||
			bidList[0].getPrice().Lt(price) || askList[0].getPrice().Gt(price)) {
			break
		}

        ExecuteOrder(price, askList[0], bidList)
        clearZeroOrderList(askList)
	}
}

// Given price, execute order and againstOrders inplace
function ExecuteOrder(price, order, againstOrders) {
    for (let i = 0; i < againstOrders.length; i++) {
        if (againstOrders[i].getSide() === OrderSide.buy) {
            if (againstOrders[i].getPrice().Lt(price)) {
                break
            }
        } else {
            if (againstOrders[i].getPrice().Gt(price)) {
                break
            }
        }

        let minAmount = againstOrders[i].getAmount()
        if (order.getAmount().Lt(againstOrders[i].getAmount())) {
            minAmount = order.getAmount();
        }

        order.deal(againstOrders[i], minAmount, price)
        clearZeroOrderList(againstOrders)

        if (order.getAmount().IsZero()) {
            break
        }
    }
}

// parameters: (U256, U256, U256, Order[], Order[])
// return: U256
function GetExecutionPrice(highPrice, midPrice, lowPrice, bidList, askList) {
    let orderList = bidList.concat(askList)
    let ppList = createPricePointList(orderList)
    accumulateForPricePointList(ppList)
    return calculateExecutionPrice(highPrice, midPrice, lowPrice, ppList)
}


function clearZeroOrderList(orderList) {
    while (orderList.length > 0 && orderList[0].getAmount().IsZero()) {
        orderList.shift()
    }
}

// ----------------------------------------------------------------


// return negative number if a has more priority than b
function pricePointComparator(a, b) {
    if (a.executionAmount.Gt(b.executionAmount)) {
        return -1
    } else if (a.executionAmount.Lt(b.executionAmount)) {
        return 1
    } else if (a.absImbalance.Lt(b.absImbalance)) {
        return -1
    } else if (a.absImbalance.Gt(b.absImbalance)) {
        return 1;
    } else {
        return a.price.Gt(b.price)
    }
}

// return PricePoint[]
function createPricePointList(orderList) {
    // price point array
    let ppList = []
    // string -> number
    let ppMap = new Map();
    for (let i = 0; i < orderList.length; i++) {
        const k = orderList[i].getPrice().String()
        let offset = ppMap.get(k)
        if (offset === undefined) {
            offset = ppList.length
            ppMap.set(k, offset)
            ppList.push({
                price: orderList[i].getPrice(),
                accumulatedAskAmount: U256(0),
                askAmount: U256(0),
                accumulatedBidAmount: U256(0),
                bidAmount: U256(0),
                executionAmount: null,
                imbalance: null,
                absImbalance: null,
            })
        }

        if (orderList[i].getSide() === OrderSide.sell) {
            ppList[offset].askAmount = ppList[offset].askAmount.Add(orderList[i].getAmount())
        } else {
            // buy
            ppList[offset].bidAmount = ppList[offset].bidAmount.Add(orderList[i].getAmount())
        }
    }
    return ppList
}


// return: U256
function calculateExecutionPrice(highPrice, midPrice, lowPrice, ppList) {
    // 1. sort price point list
    ppList.sort(pricePointComparator)

    // 2. if ppList has no same executionAmount point, use the price of the max executionAmount
    let ppListSameEA = []
    ppListSameEA.push(ppList[0])
    for (let i = 1; i < ppList.length; i++) {
        if (ppList[i].executionAmount.Equal(ppList[0].executionAmount)) {
            ppListSameEA.push(ppList[i])
        } else {
            break
        }
    }
    if (ppListSameEA.length === 1) {
        return ppListSameEA[0].price
    }

    // 3. for same executionAmount points, use the price of the smallest absImbalance
    ppListSameEA.sort(function (a, b) {
        if (a.absImbalance.Lt(b.absImbalance)) {
            return -1
        }
        return 1
    })

    let ppListSameImbalance = []
    ppListSameImbalance.push(ppListSameEA[0])
    for (let i = 1; i < ppListSameEA.length; i++) {
        if (ppListSameEA[i].absImbalance.Equal(ppListSameEA[0].absImbalance)) {
            ppListSameImbalance.push(ppList[i])
        } else {
            break
        }
    }
    if (ppListSameImbalance.length === 1) {
        return ppListSameImbalance[0].price
    }

    // 4. for same absImbalance points, consider the market pressure
    let allImbalanceIsNegative = true
    let allImbalanceIsPositive = true
    let ppWithHighestPrice = ppListSameImbalance[0]
    let ppWithLowestPrice = ppListSameImbalance[ppListSameImbalance.length-1]
    let ppWithMiddlePrice = ppListSameImbalance[ppListSameImbalance.length/2]
    const midPriceIsZero = midPrice.Equal(U256(0))
    const allPriceLargerThanHigh = ppWithLowestPrice.price.Gt(highPrice) && !midPriceIsZero
    const allPriceSmallerThanHigh = ppWithHighestPrice.price.Lt(highPrice) && !midPriceIsZero
    const allPriceLargerThanLow = ppWithLowestPrice.price.Gt(lowPrice) && !midPriceIsZero
    const allPriceSmallerThanLow = ppWithHighestPrice.price.Lt(lowPrice) && !midPriceIsZero
    if (midPriceIsZero) {
        return ppWithMiddlePrice.price
    }

    for (let i = 0; i < ppListSameImbalance.length; i++) {
        if (ppListSameImbalance[i].imbalance.Sign() === -1) {
            allImbalanceIsPositive = false
        }
        if (ppListSameImbalance[i].imbalance.Sign() === 1) {
            allImbalanceIsNegative = false
        }
    }

    if (allImbalanceIsPositive) {
        // with more buyer, we want higher price
        if (allPriceSmallerThanHigh) {
            return ppWithHighestPrice.price
        } else if (allPriceLargerThanHigh) {
            return ppWithLowestPrice.price
        } else {
            return highPrice
        }
    } else if (allImbalanceIsNegative) {
        // with more seller, we want lower price
        if (allPriceSmallerThanLow) {
            return ppWithHighestPrice.price
        } else if (allPriceLargerThanLow) {
            return ppWithLowestPrice.price
        } else {
            return lowPrice
        }
    } else {
        if (ppWithHighestPrice.price.Lt(midPrice)) {
            return ppWithHighestPrice.price
        } else if (ppWithLowestPrice.price.Gt(midPrice)) {
            return ppWithLowestPrice.price
        } else {
            return midPrice
        }
    }
}


function accumulateForPricePointList(ppList) {
    ppList.sort(function (a, b) {
        if (a.price.Gt(b.price)) {
            return -1
        }
        return 1
    });

    let accumulatedBidAmount = U256(0)
    for (let i = 0; i < ppList.length; i++) {
        accumulatedBidAmount = accumulatedBidAmount.Add(ppList[i].bidAmount)
        ppList[i].accumulatedBidAmount = accumulatedBidAmount
    }

    let accumulatedAskAmount = U256(0)
    for (let i = ppList.length - 1; i >= 0; i--) {
        accumulatedAskAmount = accumulatedAskAmount.Add(ppList[i].askAmount)
        ppList[i].accumulatedAskAmount = accumulatedAskAmount
    }

    for (let i = 0; i < ppList.length; i++) {
        ppList[i].executionAmount = ppList[i].accumulatedAskAmount
        if (ppList[i].accumulatedBidAmount.Lt(ppList[i].accumulatedAskAmount)) {
            ppList[i].executionAmount = ppList[i].accumulatedBidAmount
        }

        ppList[i].imbalance = ppList[i].accumulatedBidAmount.ToS256().Sub(ppList[i].accumulatedAskAmount.ToS256())
        ppList[i].absImbalance = ppList[i].imbalance.Abs().ToU256()
    }
}


// ----------------------------------------------------------------

function newOrderForTest(price, height, totalAmount, side, owner) {
    return new Order(U256(price), height, U256(totalAmount), side, owner)
}

function testMatch() {
    let order1 = newOrderForTest(100, 1, 150, OrderSide.buy, 'buyer1')
    let order2 = newOrderForTest(98, 1, 150, OrderSide.buy, 'buyer2')
    let bidList = [order1, order2]
    let order3 = newOrderForTest(98, 1, 250, OrderSide.sell, 'seller1')
    let order4 = newOrderForTest(97, 1, 50, OrderSide.sell, 'seller2')
    let askList = [order3, order4]

    Match(U256(100), U256(100), U256(100), bidList, askList)
}

function testHex2Number() {
    let hexList = ['0x63ae4736', '0x63ae4735', '0x63ae4734']
    Printf('min: %v\n', findMinHex(hexList))
}


function testOracle() {
    // const sbchURLs = ['https://rpc.smartbch.org', 'https://sbch-mainnet.paralinker.com/api/v1/4fd540be7cf14c437786be6415822325', 'https://smartbch.greyh.at', 'https://smartbch.fountainhead.cash/mainnet']
    const sbchURLs = ['https://sbch-mainnet.paralinker.com/api/v1/4fd540be7cf14c437786be6415822325']
    const bscURLs = ['https://bsc-mainnet.paralinker.com/api/v1/81cfef4b310965726b5326afb51ff093']
    const oracle = new Oracle()
    oracle.setChain('0x2710', sbchURLs)
    oracle.setChain('0x38', bscURLs)
    Printf('oracle: %v\n', oracle)

    let [latestHeights, latestTimestamp] = oracle.getLatestHeightFromMultiChains()
    Printf('latestHeights: %v\n', latestHeights)
    Printf('latestTimestamp: %v\n', latestTimestamp)

    let blockInfos = oracle.getNewGloballyConfirmedBlocks(latestHeights, latestTimestamp)
    Printf('globally-confirmed blocks: %v\n', blockInfos)

    let info = oracle.getNewBlockForProcessing(blockInfos)
    Printf('newBlock: %v\n', info)
    Printf('rest blockInfos: %v\n', blockInfos)
    Printf('processed blockInfos: %v\n', oracle.getProcessedHeights())

    const uncheckedHeights = ['0x91feb2', '0x1ac8778']
    const checkPointTimestamp = '0x645b5cd9'
    let ok = oracle.isAllGloballyConfirmed(uncheckedHeights, checkPointTimestamp)
    Printf('all globally-confirmed: %v\n', ok)
}

// testMatch()
// testHex2Number()
testOracle()