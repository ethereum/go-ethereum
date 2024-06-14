import type { EtherscanConfig, SourcifyConfig } from "../types";

import "hardhat/types/config";

declare module "hardhat/types/config" {
  interface HardhatUserConfig {
    etherscan?: Partial<EtherscanConfig>;
    sourcify?: Partial<SourcifyConfig>;
  }

  interface HardhatConfig {
    etherscan: EtherscanConfig;
    sourcify: SourcifyConfig;
  }
}
