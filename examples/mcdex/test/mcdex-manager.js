const {
  time,
  loadFixture,
} = require("@nomicfoundation/hardhat-network-helpers");
const { anyValue } = require("@nomicfoundation/hardhat-chai-matchers/withArgs");
const { expect } = require("chai");
const { ethers} = require("hardhat");

describe("MCDEXManager", function () {
  let allowance = 100;

  async function deployMCDEXManager() {
    const [account1, account2] = await ethers.getSigners();

    const MCDEXManager = await ethers.getContractFactory("MCDEXManager");
    const manager = await MCDEXManager.deploy();

    const TestERC20 = await ethers.getContractFactory("TestERC20");
    const myToken = await TestERC20.deploy('MYT', 100000000, 8);
    await myToken.deployed();

    await myToken.connect(account1).approve(manager.address, allowance);
    await myToken.connect(account2).approve(manager.address, allowance);

    return {manager, myToken, account1, account2};
  }

  describe("Deposit", function () {
    it("OK", async function () {
      const { manager, myToken, account1, account2 } = await loadFixture(deployMCDEXManager);
      await expect(manager.connect(account1).deposit(account1.address, concatAddressAmount(myToken.address, 100)))
          .to.emit(manager, 'Deposit').withArgs(account1.address, concatAddressAmount(myToken.address, 100));

      expect(await manager.loadWallet(myToken.address, account1.address)).to.equal(100);
    });

    it("ERC20: transfer amount exceeds balance", async function () {
      const { manager, myToken, account1, account2 } = await loadFixture(deployMCDEXManager);

      await expect(manager.connect(account2).deposit(account2.address, concatAddressAmount(myToken.address, 100)))
          .to.revertedWith('ERC20: transfer amount exceeds balance');
    });
  });
});

function concatAddressAmount(address, amount) {
  const n = BigInt(address) << 96n | BigInt(amount);
  return '0x' + n.toString(16);
}

function wrapAmount(amount) {
  const n = BigInt(amount);
  return '0x' + n.toString(16);
}