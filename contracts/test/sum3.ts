import {
    time,
    loadFixture,
} from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { anyValue } from "@nomicfoundation/hardhat-chai-matchers/withArgs";
import { expect } from "chai";
import { ethers } from "hardhat";

describe("sum3", function() {
    it("Should properly calculate sum of 3 numbers", async function () {
        const ExampleSum3 = await ethers.getContractFactory("ExampleSum3")
        const exampleSum3 = await ExampleSum3.deploy();

        const array = [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,3]
        const uint8Array = new Uint8Array(array);
        console.log(await exampleSum3.sum3(uint8Array));

        expect(await exampleSum3.sum3(uint8Array)).to.equal("0x0000000000000000000000000000000000000000000000000000000000000003");
    })
})
