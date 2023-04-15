
// expected input format {taskID: <base64str>, records: [<base64str>]} 
// each record 20byte msg.sender, 32byte amount, other: encryptedReceiver

// taskID: 160b coinType, 40b due time, 8b amount (3b 1~8, 5b decimals), 40b nonce, 8b zeros

const PREFIX = (new TextEncoder('utf-8')).encode("\x19Ethereum Signed Message:\n32");

const input = JSON.parse(CALLDATA);

const taskID = B64ToBuf(input.taskID);
let hash = new Uint8Array(taskID); // make a clone
const amount8 = hash[25];
const amountBase = U256(1+(amount8>>5)).Mul(U256(10).Exp(U256(amount8&0x1F)));

let receiverList = [];
let totalAmount = U256(0);
for(const [index, recordB64] of input.records) {
	const record = B64ToBuf(recordB64);
	hash = new Uint8Array(Keccak256(hash, record));
	hash[0] = 0; // to uint248
	const amount = BufToU256(record.slice(20, 20+32));
	const [receiver, ok] = KEY.Decrypt(recordB64.slice(20+32));
	if(ok) {
		receiverList.push(new Uint8Array(receiver));
	}
	totalAmount = totalAmount.Add(amount);
}
const totalBase = amountBase.Mul(U256(receiverList.length));
const reward = totalAmount.Sub(totalBase);
const fee = reward.Div(100); // 1% of reward used as fee
const amount = totalAmount.Sub(fee).Div(U256(receiverList.length));
receiverList.sort((a, b) => BufCompare(a, b)); // to reorder
const h = Keccak256(taskID, hash, amount, fee, ...receiverList);
const h2 = Keccak256(PREFIX, h);
const sig = KEY.Sign(h2);
const out = {
	sig: sig, 
	amount: amount.ToHex(),
	fee: fee.ToHex(),
	receivers: [],
};
for(const receiver of receiverList) {
	out.receivers.push(BufToHex(receiver));
}

JSON.stringify(out);
