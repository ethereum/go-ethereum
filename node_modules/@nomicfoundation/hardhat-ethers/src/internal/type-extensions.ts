import type { ethers } from "ethers";
import type {
  FactoryOptions as FactoryOptionsT,
  getContractFactory as getContractFactoryT,
  HardhatEthersHelpers,
  Libraries as LibrariesT,
} from "../types";

import "hardhat/types/runtime";

declare module "hardhat/types/runtime" {
  interface HardhatRuntimeEnvironment {
    // We omit the ethers field because it is redundant.
    ethers: typeof ethers & HardhatEthersHelpers;
  }

  // Beware, adding new types to any hardhat type submodule is not a good practice in a Hardhat plugin.
  // Doing so increases the risk of a type clash with another plugin.
  // Removing any of these three types is a breaking change.
  type Libraries = LibrariesT;
  type FactoryOptions = FactoryOptionsT;
  // eslint-disable-next-line @typescript-eslint/naming-convention
  type getContractFactory = typeof getContractFactoryT;
}
