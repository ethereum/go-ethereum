import type {
  Artifacts,
  BoundExperimentalHardhatNetworkMessageTraceHook,
  EIP1193Provider,
  EthereumProvider,
  HardhatConfig,
  HardhatNetworkConfig,
  HDAccountsUserConfig,
  HttpNetworkAccountsUserConfig,
  HttpNetworkConfig,
  NetworkConfig,
  ProviderExtender,
} from "../../../types";

import type {
  ForkConfig,
  MempoolOrder,
} from "../../hardhat-network/provider/node-types";
import type * as DiskCacheT from "../../hardhat-network/provider/utils/disk-cache";
import { HARDHAT_NETWORK_NAME } from "../../constants";
import { parseDateString } from "../../util/date";

import { normalizeHardhatNetworkAccountsConfig } from "./util";

export function isHDAccountsConfig(
  accounts?: HttpNetworkAccountsUserConfig
): accounts is HDAccountsUserConfig {
  return accounts !== undefined && Object.keys(accounts).includes("mnemonic");
}

function isResolvedHttpNetworkConfig(
  netConfig: Partial<NetworkConfig>
): netConfig is HttpNetworkConfig {
  return "url" in netConfig;
}

// This function is let's you import a provider dynamically in a pretty
// type-safe way.
// `ProviderNameT` and `name` must be the same literal string. TS enforces it.
// `ModuleT` and `filePath` must also be the same, but this is not enforced.
function importProvider<ModuleT, ProviderNameT extends keyof ModuleT>(
  filePath: string,
  name: ProviderNameT
): ModuleT[ProviderNameT] {
  const mod = require(filePath);
  return mod[name];
}

export async function createProvider(
  config: HardhatConfig,
  networkName: string,
  artifacts?: Artifacts,
  experimentalHardhatNetworkMessageTraceHooks: BoundExperimentalHardhatNetworkMessageTraceHook[] = [],
  extenders: ProviderExtender[] = []
): Promise<EthereumProvider> {
  let eip1193Provider: EIP1193Provider;
  const networkConfig = config.networks[networkName];
  const paths = config.paths;

  if (networkName === HARDHAT_NETWORK_NAME) {
    const hardhatNetConfig = networkConfig as HardhatNetworkConfig;

    const { createHardhatNetworkProvider } = await import(
      "../../hardhat-network/provider/provider"
    );

    let forkConfig: ForkConfig | undefined;

    if (
      hardhatNetConfig.forking?.enabled === true &&
      hardhatNetConfig.forking?.url !== undefined
    ) {
      forkConfig = {
        jsonRpcUrl: hardhatNetConfig.forking?.url,
        blockNumber: hardhatNetConfig.forking?.blockNumber,
        httpHeaders: hardhatNetConfig.forking.httpHeaders,
      };
    }

    const accounts = normalizeHardhatNetworkAccountsConfig(
      hardhatNetConfig.accounts
    );

    const { getForkCacheDirPath } =
      require("../../hardhat-network/provider/utils/disk-cache") as typeof DiskCacheT;

    eip1193Provider = await createHardhatNetworkProvider(
      {
        chainId: hardhatNetConfig.chainId,
        networkId: hardhatNetConfig.chainId,
        hardfork: hardhatNetConfig.hardfork,
        blockGasLimit: hardhatNetConfig.blockGasLimit,
        initialBaseFeePerGas: hardhatNetConfig.initialBaseFeePerGas,
        minGasPrice: hardhatNetConfig.minGasPrice,
        throwOnTransactionFailures: hardhatNetConfig.throwOnTransactionFailures,
        throwOnCallFailures: hardhatNetConfig.throwOnCallFailures,
        automine: hardhatNetConfig.mining.auto,
        intervalMining: hardhatNetConfig.mining.interval,
        // This cast is valid because of the config validation and resolution
        mempoolOrder: hardhatNetConfig.mining.mempool.order as MempoolOrder,
        chains: hardhatNetConfig.chains,
        coinbase: hardhatNetConfig.coinbase,
        genesisAccounts: accounts,
        allowUnlimitedContractSize: hardhatNetConfig.allowUnlimitedContractSize,
        allowBlocksWithSameTimestamp:
          hardhatNetConfig.allowBlocksWithSameTimestamp ?? false,
        initialDate:
          hardhatNetConfig.initialDate !== undefined
            ? parseDateString(hardhatNetConfig.initialDate)
            : undefined,
        experimentalHardhatNetworkMessageTraceHooks,
        forkConfig,
        forkCachePath:
          paths !== undefined ? getForkCacheDirPath(paths) : undefined,
        enableTransientStorage:
          hardhatNetConfig.enableTransientStorage ?? false,
      },
      {
        enabled: hardhatNetConfig.loggingEnabled,
      },
      artifacts
    );
  } else {
    const HttpProvider = importProvider<
      typeof import("./http"),
      "HttpProvider"
    >("./http", "HttpProvider");
    const httpNetConfig = networkConfig as HttpNetworkConfig;

    eip1193Provider = new HttpProvider(
      httpNetConfig.url!,
      networkName,
      httpNetConfig.httpHeaders,
      httpNetConfig.timeout
    );
  }

  let wrappedProvider = eip1193Provider;

  for (const extender of extenders) {
    wrappedProvider = await extender(wrappedProvider, config, networkName);
  }

  wrappedProvider = applyProviderWrappers(
    wrappedProvider,
    networkConfig,
    extenders
  );

  const BackwardsCompatibilityProviderAdapter = importProvider<
    typeof import("./backwards-compatibility"),
    "BackwardsCompatibilityProviderAdapter"
  >("./backwards-compatibility", "BackwardsCompatibilityProviderAdapter");

  return new BackwardsCompatibilityProviderAdapter(wrappedProvider);
}

export function applyProviderWrappers(
  provider: EIP1193Provider,
  netConfig: Partial<NetworkConfig>,
  extenders: ProviderExtender[]
): EIP1193Provider {
  // These dependencies are lazy-loaded because they are really big.
  const LocalAccountsProvider = importProvider<
    typeof import("./accounts"),
    "LocalAccountsProvider"
  >("./accounts", "LocalAccountsProvider");
  const HDWalletProvider = importProvider<
    typeof import("./accounts"),
    "HDWalletProvider"
  >("./accounts", "HDWalletProvider");
  const FixedSenderProvider = importProvider<
    typeof import("./accounts"),
    "FixedSenderProvider"
  >("./accounts", "FixedSenderProvider");
  const AutomaticSenderProvider = importProvider<
    typeof import("./accounts"),
    "AutomaticSenderProvider"
  >("./accounts", "AutomaticSenderProvider");

  const AutomaticGasProvider = importProvider<
    typeof import("./gas-providers"),
    "AutomaticGasProvider"
  >("./gas-providers", "AutomaticGasProvider");
  const FixedGasProvider = importProvider<
    typeof import("./gas-providers"),
    "FixedGasProvider"
  >("./gas-providers", "FixedGasProvider");
  const AutomaticGasPriceProvider = importProvider<
    typeof import("./gas-providers"),
    "AutomaticGasPriceProvider"
  >("./gas-providers", "AutomaticGasPriceProvider");
  const FixedGasPriceProvider = importProvider<
    typeof import("./gas-providers"),
    "FixedGasPriceProvider"
  >("./gas-providers", "FixedGasPriceProvider");
  const ChainIdValidatorProvider = importProvider<
    typeof import("./chainId"),
    "ChainIdValidatorProvider"
  >("./chainId", "ChainIdValidatorProvider");

  if (isResolvedHttpNetworkConfig(netConfig)) {
    const accounts = netConfig.accounts;

    if (Array.isArray(accounts)) {
      provider = new LocalAccountsProvider(provider, accounts);
    } else if (isHDAccountsConfig(accounts)) {
      provider = new HDWalletProvider(
        provider,
        accounts.mnemonic,
        accounts.path,
        accounts.initialIndex,
        accounts.count,
        accounts.passphrase
      );
    }

    // TODO: Add some extension mechanism for account plugins here
  }

  if (netConfig.from !== undefined) {
    provider = new FixedSenderProvider(provider, netConfig.from);
  } else {
    provider = new AutomaticSenderProvider(provider);
  }

  if (netConfig.gas === undefined || netConfig.gas === "auto") {
    provider = new AutomaticGasProvider(provider, netConfig.gasMultiplier);
  } else {
    provider = new FixedGasProvider(provider, netConfig.gas);
  }

  if (netConfig.gasPrice === undefined || netConfig.gasPrice === "auto") {
    // If you use a LocalAccountsProvider or HDWalletProvider, your transactions
    // are signed locally. This requires having all of their fields available,
    // including the gasPrice / maxFeePerGas & maxPriorityFeePerGas.
    //
    // We never use those providers when using Hardhat Network, but sign within
    // Hardhat Network itself. This means that we don't need to provide all the
    // fields, as the missing ones will be resolved there.
    //
    // Hardhat Network handles this in a more performant way, so we don't use
    // the AutomaticGasPriceProvider for it unless there are provider extenders.
    // The reason for this is that some extenders (like hardhat-ledger's) might
    // do the signing themselves, and that needs the gas price to be set.
    if (isResolvedHttpNetworkConfig(netConfig) || extenders.length > 0) {
      provider = new AutomaticGasPriceProvider(provider);
    }
  } else {
    provider = new FixedGasPriceProvider(provider, netConfig.gasPrice);
  }

  if (
    isResolvedHttpNetworkConfig(netConfig) &&
    netConfig.chainId !== undefined
  ) {
    provider = new ChainIdValidatorProvider(provider, netConfig.chainId);
  }

  return provider;
}
