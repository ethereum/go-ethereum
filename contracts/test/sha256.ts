import {
    time,
    loadFixture,
} from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { anyValue } from "@nomicfoundation/hardhat-chai-matchers/withArgs";
import { expect } from "chai";
import { ethers } from "hardhat";

describe("SHA256", function() {
    it("Should properly calculate hash", async function () {
        const ExampleSHA256 = await ethers.getContractFactory("ExampleSHA256")
        const exampleSHA256 = await ExampleSHA256.deploy();

        expect(await exampleSHA256.hashSha256(42)).to.equal("0x0a28e9ffef0073f9a6a674cf57ee77307f38f0f1bebb087888d9011ed0eeefdf");
    })
})
