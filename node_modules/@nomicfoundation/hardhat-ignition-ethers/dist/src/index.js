"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
require("@nomicfoundation/hardhat-ethers");
require("@nomicfoundation/hardhat-ignition");
const config_1 = require("hardhat/config");
const plugins_1 = require("hardhat/plugins");
require("./type-extensions");
/**
 * Add an `ignition` object to the HRE.
 */
(0, config_1.extendEnvironment)((hre) => {
    if (hre.ignition !== undefined &&
        hre.ignition.type !== "stub" &&
        hre.ignition.type !== "ethers") {
        throw new plugins_1.HardhatPluginError("hardhat-ignition-ethers", `Found ${hre.ignition.type} and ethers, but only one Hardhat Ignition extension plugin can be used at a time.`);
    }
    hre.ignition = (0, plugins_1.lazyObject)(() => {
        const { EthersIgnitionHelper } = require("./ethers-ignition-helper");
        return new EthersIgnitionHelper(hre);
    });
});
//# sourceMappingURL=index.js.map