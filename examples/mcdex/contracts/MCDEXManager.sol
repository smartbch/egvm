// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.18;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
// Uncomment this line to use console.log
// import "hardhat/console.sol";


contract MCDEXManager {
    address constant private SEP206Contract = address(uint160(0x2711));

    mapping (address => mapping (address => uint)) walletMap; // Coin => (EOA => Balance)


    event Deposit(address indexed owner, uint indexed coin_amount);

    function safeReceive(address coin, uint amount) internal returns (uint) {
        uint realAmount = amount;
        if(coin == SEP206Contract) {
            require(msg.value == amount, "value-mismatch");
        } else {
            require(msg.value == 0, "dont-send-bch");
            uint oldBalance = IERC20(coin).balanceOf(address(this));
            IERC20(coin).transferFrom(msg.sender, address(this), uint(amount));
            uint newBalance = IERC20(coin).balanceOf(address(this));
            realAmount = newBalance - oldBalance;
        }
        return realAmount;
    }

    function safeTransfer(address coin, address receiver, uint amount) internal {
        if(amount == 0) {
            return;
        }
        (bool success, bytes memory data) = coin.call(
            abi.encodeWithSignature("transfer(address,uint256)", receiver, amount));
        bool ret = abi.decode(data, (bool));
        require(success && ret, "transfer-failed");
    }

    function saveWallet(address token, address owner, uint balance) internal {
        walletMap[token][owner] = balance;
    }

    function loadWallet(address token, address owner) public view returns (uint) {
        return walletMap[token][owner];
    }


    function deposit(address owner, uint coin_amount) public payable {
        address coin = address(uint160(coin_amount >>96));
        uint amount = uint96(coin_amount);
        safeReceive(coin, amount);
        uint balance = loadWallet(coin, owner);
        saveWallet(coin, owner, balance + amount);
        emit Deposit(owner, coin_amount);
    }

    function withdraw(uint coin_amount) public {
        address coin = address(uint160(coin_amount >>96));
        uint amount = uint96(coin_amount);
        uint balance = loadWallet(coin, msg.sender);
        require(balance >= amount, "not-enough-balance");
        safeTransfer(coin, msg.sender, amount);
        saveWallet(coin, msg.sender, balance - amount);
    }
}

