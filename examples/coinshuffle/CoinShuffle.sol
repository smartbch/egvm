// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.15;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

contract CoinShuffle {
	uint constant MAX_WAITING_TIME = 2 days;
	uint constant MAX_REWARD_RATIO = 12; //12%
	uint constant MAX_INPUTS = 32;
	string constant private PREFIX = "\x19Ethereum Signed Message:\n32";

	struct TaskStatus {
		uint8 count;
		uint248 hash;
	}

	mapping (uint248 => TaskStatus) public taskMap;
	address immutable CONTROLLER;

	event CreateTask(uint indexed taskID);
	event JoinTask(uint indexed taskID, address indexed sender, uint amount, bytes encryptedReceiver);
	event FinishTask(uint indexed taskID);

	constructor(address ctrl) {
		CONTROLLER = ctrl;
	}

        // taskID: 160b coinType, 40b due time, 8b amount (3b 1~8, 5b decimals), 40b nonce

	function expandAmount(uint8 amount) public pure returns (uint) {
		return (1+(amount>>5))*(10**(amount&0x1F));
	}

	function createTask(uint248 taskID) public {
		uint dueTime = uint40(taskID>>48);
		require(dueTime < block.timestamp + MAX_WAITING_TIME, "due time too late");
		require(taskMap[taskID].hash == 0, "task already exist");
		require(uint8(taskID) == 0, "non-zero lsb");
		taskMap[taskID] = TaskStatus({count: 0, hash: taskID});
		emit CreateTask(taskID);
	}

	function joinTask(uint248 taskID, uint amount, bytes calldata encryptedReceiver) public {
		TaskStatus memory status = taskMap[taskID];
		if(status.hash == 0) {
			createTask(taskID);
			status = TaskStatus({count: 0, hash: taskID});
		}
		uint dueTime = uint40(taskID >> 48);
		require(block.timestamp < dueTime, "too late");
		uint amountBase = expandAmount(uint8(taskID>>40));
		require(amount >= amountBase, "amount not enough");
		require(amount < amountBase * (100+MAX_REWARD_RATIO) / 100, "amount too much");
		IERC20(address(uint160(taskID>>88))).transferFrom(msg.sender, address(this), amount);
		require(status.count < MAX_INPUTS, "too many inputs");
		status.count++;
		bytes32 newHash = keccak256(abi.encodePacked(uint(status.hash), msg.sender,
							     amount, encryptedReceiver));
		status.hash = uint248(uint(newHash));
		taskMap[taskID] = status;
		emit JoinTask(taskID, msg.sender, amount, encryptedReceiver);
	}

	function finishTask(uint248 taskID, uint amount, uint fee, address[] calldata receivers,
			    uint8 v, bytes32 r, bytes32 s) public {
		bytes32 h = keccak256(abi.encodePacked(taskID, uint(taskMap[taskID].hash),
						       amount, fee, receivers));
		bytes32 h2 = keccak256(abi.encodePacked(PREFIX, h));
		require(CONTROLLER == ecrecover(h2, v, r, s), "incorrect signature");
		delete taskMap[taskID];
		address coinAddr = address(uint160(taskID>>88));
		IERC20(coinAddr).transfer(msg.sender, fee);
		for(uint i=0; i<receivers.length; i++) {
			IERC20(coinAddr).transfer(receivers[i], amount);
		}
		emit FinishTask(taskID);
	}
}
