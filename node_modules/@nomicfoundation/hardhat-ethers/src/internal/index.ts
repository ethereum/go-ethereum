import type EthersT from "ethers";
import type { HardhatEthersProvider as HardhatEthersProviderT } from "./hardhat-ethers-provider";

import { extendEnvironment } from "hardhat/config";
import { lazyObject } from "hardhat/plugins";

import {
  getContractAt,
  getContractAtFromArtifact,
  getContractFactory,
  getContractFactoryFromArtifact,
  getImpersonatedSigner,
  getSigner,
  getSigners,
  deployContract,
} from "./helpers";
import "./type-extensions";

extendEnvironment((hre) => {
  hre.ethers = lazyObject(() => {
    const { ethers } = require("ethers") as typeof EthersT;
    const { HardhatEthersProvider } = require("./hardhat-ethers-provider") as {
      HardhatEthersProvider: typeof HardhatEthersProviderT;
    };

    const provider = new HardhatEthersProvider(
      hre.network.provider,
      hre.network.name
    );

    return {
      ...ethers,

      provider,

      getSigner: (address: string) => getSigner(hre, address),
      getSigners: () => getSigners(hre),
      getImpersonatedSigner: (address: string) =>
        getImpersonatedSigner(hre, address),
      // We cast to any here as we hit a limitation of Function#bind and
      // overloads. See: https://github.com/microsoft/TypeScript/issues/28582
      getContractFactory: getContractFactory.bind(null, hre) as any,
      getContractFactoryFromArtifact: (...args) =>
        getContractFactoryFromArtifact(hre, ...args),
      getContractAt: (...args) => getContractAt(hre, ...args),
      getContractAtFromArtifact: (...args) =>
        getContractAtFromArtifact(hre, ...args),
      deployContract: deployContract.bind(null, hre) as any,
    };
  });
});
