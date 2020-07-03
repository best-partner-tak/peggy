import chai from "chai";
import { ethers } from "@nomiclabs/buidler";
import { solidity } from "ethereum-waffle";

import { TestERC20 } from "../typechain/TestERC20";
import { SigningTest } from "../typechain/SigningTest";
import { Signer } from "ethers";

chai.use(solidity);
const { expect } = chai;

describe("Diagnostic tests", function() {
  it("Coin test", async function() {
    const TestERC20 = await ethers.getContractFactory("TestERC20");
    const testERC20 = (await TestERC20.deploy()) as TestERC20;

    const signers = await ethers.getSigners();

    await testERC20.deployed();

    await testERC20.connect(signers[1]).transfer(await signers[2].getAddress(), 99);

    await testERC20
      .connect(signers[1])
      .transfer(await signers[2].getAddress(), 9999999999);
  });
});

describe("signing test", () => {
  it("Signing test simple", async function() {
    const signers = await ethers.getSigners();

    const SigningTest = await ethers.getContractFactory("SigningTest");
    const st = (await SigningTest.deploy()) as SigningTest;

    await st.deployed();

    const signerAddress = await signers[2].getAddress();
    const theHash = ethers.utils.formatBytes32String("foo");
    const { v, r, s } = ethers.utils.splitSignature(
      await signers[2].signMessage(ethers.utils.arrayify(theHash))
    );

    ethers;

    st.checkSignature(signerAddress, theHash, v!, r, s);
  });

  it("signs right w/ function", async () => {
    const SigningTest = await ethers.getContractFactory("SigningTest");
    const signingTest = (await SigningTest.deploy()) as SigningTest;

    const signers = await ethers.getSigners();

    const data = ethers.utils.formatBytes32String("hello");

    let theHash = ethers.utils.solidityKeccak256(["bytes32"], [data]);

    const { v, r, s } = await signHash([signers[1]], theHash);

    signingTest.checkSignature(
      await signers[1].getAddress(),
      theHash,
      v[0],
      r[0],
      s[0]
    );
  });
});

async function signHash(signers: Signer[], hash: string) {
  let v: number[] = [];
  let r: string[] = [];
  let s: string[] = [];

  for (let i = 0; i < signers.length; i = i + 1) {
    const sig = await signers[i].signMessage(ethers.utils.arrayify(hash));
    const address = await signers[i].getAddress();

    const splitSig = ethers.utils.splitSignature(sig);
    v.push(splitSig.v!);
    r.push(splitSig.r);
    s.push(splitSig.s);
  }
  return { v, r, s };
}
