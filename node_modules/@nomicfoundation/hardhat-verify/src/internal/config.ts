import type LodashCloneDeepT from "lodash.clonedeep";
import type { HardhatConfig, HardhatUserConfig } from "hardhat/types";
import type { EtherscanConfig, SourcifyConfig } from "../types";

import chalk from "chalk";

export function etherscanConfigExtender(
  config: HardhatConfig,
  userConfig: Readonly<HardhatUserConfig>
): void {
  const defaultEtherscanConfig: EtherscanConfig = {
    apiKey: "",
    customChains: [],
    enabled: true,
  };
  const cloneDeep = require("lodash.clonedeep") as typeof LodashCloneDeepT;
  const userEtherscanConfig = cloneDeep(userConfig.etherscan);
  config.etherscan = { ...defaultEtherscanConfig, ...userEtherscanConfig };

  // check that there is no etherscan entry in the networks object, since
  // this is a common mistake made by users
  if (
    userConfig.etherscan === undefined &&
    config.networks?.etherscan !== undefined
  ) {
    console.warn(
      chalk.yellow(
        "WARNING: you have an 'etherscan' entry in your networks configuration. This is likely a mistake. The etherscan configuration should be at the root of the configuration, not within the networks object."
      )
    );
  }
}

export function sourcifyConfigExtender(
  config: HardhatConfig,
  userConfig: Readonly<HardhatUserConfig>
): void {
  const defaultSourcifyConfig: SourcifyConfig = {
    enabled: false,
    apiUrl: "https://sourcify.dev/server",
    browserUrl: "https://repo.sourcify.dev",
  };
  const cloneDeep = require("lodash.clonedeep") as typeof LodashCloneDeepT;
  const userSourcifyConfig = cloneDeep(userConfig.sourcify);
  config.sourcify = { ...defaultSourcifyConfig, ...userSourcifyConfig };
}
