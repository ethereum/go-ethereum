import { HardhatNetworkConfig } from "../../../types";
import { HardforkName } from "../../util/hardforks";
import { HARDHAT_NETWORK_NAME } from "../../constants";

export const DEFAULT_SOLC_VERSION = "0.7.3";
export const HARDHAT_NETWORK_DEFAULT_GAS_PRICE = "auto";
export const HARDHAT_NETWORK_DEFAULT_MAX_PRIORITY_FEE_PER_GAS = 1e9;
export const HARDHAT_NETWORK_DEFAULT_INITIAL_BASE_FEE_PER_GAS = 1e9;
export const HARDHAT_NETWORK_MNEMONIC =
  "test test test test test test test test test test test junk";
export const DEFAULT_HARDHAT_NETWORK_BALANCE = "10000000000000000000000";

export const defaultDefaultNetwork = HARDHAT_NETWORK_NAME;

export const defaultLocalhostNetworkParams = {
  url: "http://127.0.0.1:8545",
  timeout: 40000,
};

export const defaultHdAccountsConfigParams = {
  initialIndex: 0,
  count: 20,
  path: "m/44'/60'/0'/0",
  passphrase: "",
};

export const defaultHardhatNetworkHdAccountsConfigParams = {
  ...defaultHdAccountsConfigParams,
  mnemonic: HARDHAT_NETWORK_MNEMONIC,
  accountsBalance: DEFAULT_HARDHAT_NETWORK_BALANCE,
};

export const DEFAULT_GAS_MULTIPLIER = 1;

export const defaultHardhatNetworkParams: Omit<
  HardhatNetworkConfig,
  "gas" | "initialDate"
> = {
  hardfork: HardforkName.PRAGUE,
  blockGasLimit: 30_000_000,
  gasPrice: HARDHAT_NETWORK_DEFAULT_GAS_PRICE,
  chainId: 31337,
  throwOnTransactionFailures: true,
  throwOnCallFailures: true,
  allowUnlimitedContractSize: false,
  mining: {
    auto: true,
    interval: 0,
    mempool: {
      order: "priority",
    },
  },
  accounts: defaultHardhatNetworkHdAccountsConfigParams,
  loggingEnabled: false,
  gasMultiplier: DEFAULT_GAS_MULTIPLIER,
  minGasPrice: 0n,
  /**
   * Block numbers / timestamps were taken from:
   * https://github.com/ethereumjs/ethereumjs-monorepo/tree/master/packages/common/src/chains.ts
   *
   * To find hardfork activation blocks by timestamp, use:
   * https://api-TESTNET.etherscan.io/api?module=block&action=getblocknobytime&timestamp=TIMESTAMP&closest=before&apikey=APIKEY
   */
  chains: new Map([
    [
      1, // mainnet
      {
        hardforkHistory: new Map([
          [HardforkName.FRONTIER, 0],
          [HardforkName.HOMESTEAD, 1_150_000],
          [HardforkName.DAO, 1_920_000],
          [HardforkName.TANGERINE_WHISTLE, 2_463_000],
          [HardforkName.SPURIOUS_DRAGON, 2_675_000],
          [HardforkName.BYZANTIUM, 4_370_000],
          [HardforkName.CONSTANTINOPLE, 7_280_000],
          [HardforkName.PETERSBURG, 7_280_000],
          [HardforkName.ISTANBUL, 9_069_000],
          [HardforkName.MUIR_GLACIER, 9_200_000],
          [HardforkName.BERLIN, 1_2244_000],
          [HardforkName.LONDON, 12_965_000],
          [HardforkName.ARROW_GLACIER, 13_773_000],
          [HardforkName.GRAY_GLACIER, 15_050_000],
          [HardforkName.MERGE, 15_537_394],
          [HardforkName.SHANGHAI, 17_034_870],
          [HardforkName.CANCUN, 19_426_589],
          [HardforkName.PRAGUE, 22_431_084],
        ]),
      },
    ],
    [
      5, // goerli
      {
        hardforkHistory: new Map([
          [HardforkName.ISTANBUL, 1_561_651],
          [HardforkName.BERLIN, 4_460_644],
          [HardforkName.LONDON, 5_062_605],
        ]),
      },
    ],
    [
      17000, // holesky
      {
        hardforkHistory: new Map([
          [HardforkName.MERGE, 0],
          [HardforkName.SHANGHAI, 6_698],
          [HardforkName.CANCUN, 894_732],
          [HardforkName.PRAGUE, 3_419_704],
        ]),
      },
    ],
    [
      560048, // hoodi
      {
        hardforkHistory: new Map([
          [HardforkName.MERGE, 0],
          [HardforkName.SHANGHAI, 0],
          [HardforkName.CANCUN, 0],
          [HardforkName.PRAGUE, 60_412],
        ]),
      },
    ],
    [
      11155111, // sepolia
      {
        hardforkHistory: new Map([
          [HardforkName.GRAY_GLACIER, 0],
          [HardforkName.MERGE, 1_450_409],
          [HardforkName.SHANGHAI, 2_990_908],
          [HardforkName.CANCUN, 5_187_023],
          [HardforkName.PRAGUE, 7_836_331],
        ]),
      },
    ],
    // TODO: the rest of this config is a temporary workaround,
    // see https://github.com/NomicFoundation/edr/issues/522
    [
      10, // optimism mainnet
      {
        hardforkHistory: new Map([[HardforkName.SHANGHAI, 0]]),
      },
    ],
    [
      11155420, // optimism sepolia
      {
        hardforkHistory: new Map([[HardforkName.SHANGHAI, 0]]),
      },
    ],
    [
      42161, // arbitrum one
      {
        hardforkHistory: new Map([[HardforkName.SHANGHAI, 0]]),
      },
    ],
    [
      43114, // avalanche
      {
        hardforkHistory: new Map([
          [HardforkName.SHANGHAI, 11_404_279],
          [HardforkName.CANCUN, 41_263_126],
        ]),
      },
    ],
    [
      421614, // arbitrum sepolia
      {
        hardforkHistory: new Map([[HardforkName.SHANGHAI, 0]]),
      },
    ],
  ]),
};

export const defaultHttpNetworkParams = {
  accounts: "remote" as "remote",
  gas: "auto" as "auto",
  gasPrice: "auto" as "auto",
  gasMultiplier: DEFAULT_GAS_MULTIPLIER,
  httpHeaders: {},
  timeout: 20000,
};

export const defaultMochaOptions: Mocha.MochaOptions = {
  timeout: 40000,
};

export const defaultSolcOutputSelection = {
  "*": {
    "*": [
      "abi",
      "evm.bytecode",
      "evm.deployedBytecode",
      "evm.methodIdentifiers",
      "metadata",
    ],
    "": ["ast"],
  },
};
