// This file defines the different config types.
//
// For each possible kind of config value, we have two types:
//
// One that ends with UserConfig, which represent the config as
// written in the user's config file.
//
// The other one, with the same name except without the User part, represents
// the resolved value as used during the hardhat execution.
//
// Note that while many declarations are repeated here (i.e. network types'
// fields), we don't use `extends` as that can interfere with plugin authors
// trying to augment the config types.

// Networks config

export interface NetworksUserConfig {
  hardhat?: HardhatNetworkUserConfig;

  [networkName: string]: NetworkUserConfig | undefined;
}

export type NetworkUserConfig =
  | HardhatNetworkUserConfig
  | HttpNetworkUserConfig;

export interface HardforkHistoryUserConfig {
  [hardforkName: string]: number /* block number */;
}

export interface HardhatNetworkChainUserConfig {
  hardforkHistory?: HardforkHistoryUserConfig;
}

export interface HardhatNetworkChainsUserConfig {
  [chainId: number]: HardhatNetworkChainUserConfig;
}

export interface HardhatNetworkUserConfig {
  chainId?: number;
  from?: string;
  gas?: "auto" | number;
  gasPrice?: "auto" | number;
  gasMultiplier?: number;
  initialBaseFeePerGas?: number;
  hardfork?: string;
  mining?: HardhatNetworkMiningUserConfig;
  accounts?: HardhatNetworkAccountsUserConfig;
  blockGasLimit?: number;
  minGasPrice?: number | string;
  throwOnTransactionFailures?: boolean;
  throwOnCallFailures?: boolean;
  allowUnlimitedContractSize?: boolean;
  allowBlocksWithSameTimestamp?: boolean;
  initialDate?: string;
  loggingEnabled?: boolean;
  forking?: HardhatNetworkForkingUserConfig;
  coinbase?: string;
  chains?: HardhatNetworkChainsUserConfig;
  enableTransientStorage?: boolean;
}

export type HardhatNetworkAccountsUserConfig =
  | HardhatNetworkAccountUserConfig[]
  | HardhatNetworkHDAccountsUserConfig;

export interface HardhatNetworkAccountUserConfig {
  privateKey: string;
  balance: string;
}

export interface HardhatNetworkHDAccountsUserConfig {
  mnemonic?: string;
  initialIndex?: number;
  count?: number;
  path?: string;
  accountsBalance?: string;
  passphrase?: string;
}

export interface HDAccountsUserConfig {
  mnemonic: string;
  initialIndex?: number;
  count?: number;
  path?: string;
  passphrase?: string;
}

export interface HardhatNetworkForkingUserConfig {
  enabled?: boolean;
  url: string;
  blockNumber?: number;
  httpHeaders?: { [name: string]: string };
}

export type HttpNetworkAccountsUserConfig =
  | "remote"
  | string[]
  | HDAccountsUserConfig;

export interface HttpNetworkUserConfig {
  chainId?: number;
  from?: string;
  gas?: "auto" | number;
  gasPrice?: "auto" | number;
  gasMultiplier?: number;
  url?: string;
  timeout?: number;
  httpHeaders?: { [name: string]: string };
  accounts?: HttpNetworkAccountsUserConfig;
}

export interface NetworksConfig {
  hardhat: HardhatNetworkConfig;
  localhost: HttpNetworkConfig;

  [networkName: string]: NetworkConfig;
}

export type NetworkConfig = HardhatNetworkConfig | HttpNetworkConfig;

export type HardforkHistoryConfig = Map<
  /* hardforkName */ string,
  /* blockNumber */ number
>;

export interface HardhatNetworkChainConfig {
  hardforkHistory: HardforkHistoryConfig;
}

export type HardhatNetworkChainsConfig = Map<
  /* chainId */ number,
  HardhatNetworkChainConfig
>;

export interface HardhatNetworkConfig {
  chainId: number;
  from?: string;
  gas: "auto" | number;
  gasPrice: "auto" | number;
  gasMultiplier: number;
  initialBaseFeePerGas?: number;
  hardfork: string;
  mining: HardhatNetworkMiningConfig;
  accounts: HardhatNetworkAccountsConfig;
  blockGasLimit: number;
  minGasPrice: bigint;
  throwOnTransactionFailures: boolean;
  throwOnCallFailures: boolean;
  allowUnlimitedContractSize: boolean;
  initialDate: string;
  loggingEnabled: boolean;
  forking?: HardhatNetworkForkingConfig;
  coinbase?: string;
  chains: HardhatNetworkChainsConfig;
  allowBlocksWithSameTimestamp?: boolean;
  enableTransientStorage?: boolean;
}

export type HardhatNetworkAccountsConfig =
  | HardhatNetworkHDAccountsConfig
  | HardhatNetworkAccountConfig[];

export interface HardhatNetworkAccountConfig {
  privateKey: string;
  balance: string;
}

export interface HardhatNetworkHDAccountsConfig {
  mnemonic: string;
  initialIndex: number;
  count: number;
  path: string;
  accountsBalance: string;
  passphrase: string;
}

export interface HardhatNetworkForkingConfig {
  enabled: boolean;
  url: string;
  blockNumber?: number;
  httpHeaders?: { [name: string]: string };
}

export interface HttpNetworkConfig {
  chainId?: number;
  from?: string;
  gas: "auto" | number;
  gasPrice: "auto" | number;
  gasMultiplier: number;
  url: string;
  timeout: number;
  httpHeaders: { [name: string]: string };
  accounts: HttpNetworkAccountsConfig;
}

export type HttpNetworkAccountsConfig =
  | "remote"
  | string[]
  | HttpNetworkHDAccountsConfig;

export interface HttpNetworkHDAccountsConfig {
  mnemonic: string;
  initialIndex: number;
  count: number;
  path: string;
  passphrase: string;
}

export interface HardhatNetworkMiningConfig {
  auto: boolean;
  interval: number | [number, number];
  mempool: HardhatNetworkMempoolConfig;
}

export interface HardhatNetworkMiningUserConfig {
  auto?: boolean;
  interval?: number | [number, number];
  mempool?: HardhatNetworkMempoolUserConfig;
}

export interface HardhatNetworkMempoolConfig {
  order: string; // Guaranteed at runtime to be have a valid value
}

export interface HardhatNetworkMempoolUserConfig {
  order?: string;
}

// Project paths config

export interface ProjectPathsUserConfig {
  root?: string;
  cache?: string;
  artifacts?: string;
  sources?: string;
  tests?: string;
}

export interface ProjectPathsConfig {
  root: string;
  configFile: string;
  cache: string;
  artifacts: string;
  sources: string;
  tests: string;
}

// Solidity config

// Note that the user config SolidityUserConfig is more complex than the resolved config SolidityConfig
export type SolidityUserConfig = string | SolcUserConfig | MultiSolcUserConfig;

export interface SolcUserConfig {
  version: string;
  settings?: any;
}

export interface MultiSolcUserConfig {
  compilers: SolcUserConfig[];
  overrides?: Record<string, SolcUserConfig>;
}

export interface SolcConfig {
  version: string;
  settings: any;
}

export interface SolidityConfig {
  compilers: SolcConfig[];
  overrides: Record<string, SolcConfig>;
}

// Hardhat config

export interface HardhatUserConfig {
  defaultNetwork?: string;
  paths?: ProjectPathsUserConfig;
  networks?: NetworksUserConfig;
  solidity?: SolidityUserConfig;
  mocha?: Mocha.MochaOptions;
}

export interface HardhatConfig {
  defaultNetwork: string;
  paths: ProjectPathsConfig;
  networks: NetworksConfig;
  solidity: SolidityConfig;
  mocha: Mocha.MochaOptions;
}

// Plugins config functionality

export type ConfigExtender = (
  config: HardhatConfig,
  userConfig: Readonly<HardhatUserConfig>
) => void;
