"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const config_1 = require("hardhat/config");
const plugins_1 = require("hardhat/plugins");
const helpers_1 = require("./helpers");
require("./type-extensions");
(0, config_1.extendEnvironment)((hre) => {
    hre.ethers = (0, plugins_1.lazyObject)(() => {
        const { ethers } = require("ethers");
        const { HardhatEthersProvider } = require("./hardhat-ethers-provider");
        const provider = new HardhatEthersProvider(hre.network.provider, hre.network.name);
        return {
            ...ethers,
            provider,
            getSigner: (address) => (0, helpers_1.getSigner)(hre, address),
            getSigners: () => (0, helpers_1.getSigners)(hre),
            getImpersonatedSigner: (address) => (0, helpers_1.getImpersonatedSigner)(hre, address),
            // We cast to any here as we hit a limitation of Function#bind and
            // overloads. See: https://github.com/microsoft/TypeScript/issues/28582
            getContractFactory: helpers_1.getContractFactory.bind(null, hre),
            getContractFactoryFromArtifact: (...args) => (0, helpers_1.getContractFactoryFromArtifact)(hre, ...args),
            getContractAt: (...args) => (0, helpers_1.getContractAt)(hre, ...args),
            getContractAtFromArtifact: (...args) => (0, helpers_1.getContractAtFromArtifact)(hre, ...args),
            deployContract: helpers_1.deployContract.bind(null, hre),
        };
    });
});
//# sourceMappingURL=index.js.map