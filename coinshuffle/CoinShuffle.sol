// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.15;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

contract CoinShuffle {
	uint constant MAX_WAITING_TIME = 2 days;
	uint constant MAX_REWARD_RATIO = 12; //12%
	uint constant MAX_INPUTS = 32;
	string constant private PREFIX = "\x19Ethereum Signed Message:\n32";

	mapping (uint => uint) public taskMap;
	address immutable CONTROLLER;

	event CreateTask(uint indexed taskID);
	event JoinTask(uint indexed taskID, address indexed sender, uint amount, bytes secret);
	event FinishTask(uint indexed taskID);

	constructor(address ctrl) {
		CONTROLLER = ctrl;
	}

        // taskID: 160b coinType, 40b due time, 8b amount (3b 1~8, 5b decimals), 40b nonce, 8b zeros

	function expandAmount(uint8 amount) public pure returns (uint) {
		return (1+(amount>>5))*(10**(amount&0x1F));
	}

	function createTask(uint taskID) public {
		uint dueTime = uint40(taskID>>56);
		require(dueTime < block.timestamp + MAX_WAITING_TIME, "due time too late");
		require(uint(taskMap[taskID]) == 0, "task already exist");
		require(uint8(taskID) == 0, "non-zero lsb");
		taskMap[taskID] = taskID;
		emit CreateTask(taskID);
	}

	function joinTask(uint taskID, uint amount, bytes calldata secret) public {
		uint hashAndCount = taskMap[taskID];
		if(hashAndCount == 0) {
			createTask(taskID);
			hashAndCount = taskID;
		}
		uint dueTime = uint40(taskID >> 56);
		require(block.timestamp < dueTime, "too late");
		uint amountSpec = expandAmount(uint8(taskID>>48));
		require(amount * 100 < amountSpec * (100+MAX_REWARD_RATIO), "amount too high");
		IERC20(address(uint160(taskID>>96))).transferFrom(msg.sender, address(this), amount);
		uint count = uint8(hashAndCount) + 1;
		require(count <= MAX_INPUTS, "too many inputs");
		bytes32 newHash = keccak256(abi.encodePacked(hashAndCount, msg.sender, amount, secret));
		taskMap[taskID] = (uint(newHash)&~uint(0xFF)) + count;
		emit JoinTask(taskID, msg.sender, amount, secret);
	}

	function finishTask(uint taskID, uint hash, uint amount, uint fee, address[] calldata receivers,
			    uint8 v, bytes32 r, bytes32 s) public {
		bytes32 h = keccak256(abi.encodePacked(taskID, hash, amount, fee, receivers));
		bytes32 h2 = keccak256(abi.encodePacked(PREFIX, h));
		require(CONTROLLER == ecrecover(h2, v, r, s), "incorrect signature");
		require(taskMap[taskID] == hash, "incorrect task hash");
		delete taskMap[taskID];
		address coinAddr = address(uint160(taskID>>96));
		IERC20(coinAddr).transfer(msg.sender, fee);
		for(uint i=0; i<receivers.length; i++) {
			IERC20(coinAddr).transfer(receivers[i], amount);
		}
		emit FinishTask(taskID);
	}
}
