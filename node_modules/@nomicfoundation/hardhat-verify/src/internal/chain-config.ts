import type { ChainConfig } from "../types";

// See https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md#list-of-chain-ids
export const builtinChains: ChainConfig[] = [
  {
    network: "mainnet",
    chainId: 1,
    urls: {
      apiURL: "https://api.etherscan.io/api",
      browserURL: "https://etherscan.io",
    },
  },
  {
    network: "goerli",
    chainId: 5,
    urls: {
      apiURL: "https://api-goerli.etherscan.io/api",
      browserURL: "https://goerli.etherscan.io",
    },
  },
  {
    network: "optimisticEthereum",
    chainId: 10,
    urls: {
      apiURL: "https://api-optimistic.etherscan.io/api",
      browserURL: "https://optimistic.etherscan.io/",
    },
  },
  {
    network: "bsc",
    chainId: 56,
    urls: {
      apiURL: "https://api.bscscan.com/api",
      browserURL: "https://bscscan.com",
    },
  },
  {
    network: "sokol",
    chainId: 77,
    urls: {
      apiURL: "https://blockscout.com/poa/sokol/api",
      browserURL: "https://blockscout.com/poa/sokol",
    },
  },
  {
    network: "bscTestnet",
    chainId: 97,
    urls: {
      apiURL: "https://api-testnet.bscscan.com/api",
      browserURL: "https://testnet.bscscan.com",
    },
  },
  {
    network: "xdai",
    chainId: 100,
    urls: {
      apiURL: "https://api.gnosisscan.io/api",
      browserURL: "https://gnosisscan.io",
    },
  },
  {
    network: "gnosis",
    chainId: 100,
    urls: {
      apiURL: "https://api.gnosisscan.io/api",
      browserURL: "https://gnosisscan.io",
    },
  },
  {
    network: "heco",
    chainId: 128,
    urls: {
      apiURL: "https://api.hecoinfo.com/api",
      browserURL: "https://hecoinfo.com",
    },
  },
  {
    network: "polygon",
    chainId: 137,
    urls: {
      apiURL: "https://api.polygonscan.com/api",
      browserURL: "https://polygonscan.com",
    },
  },
  {
    network: "opera",
    chainId: 250,
    urls: {
      apiURL: "https://api.ftmscan.com/api",
      browserURL: "https://ftmscan.com",
    },
  },
  {
    network: "hecoTestnet",
    chainId: 256,
    urls: {
      apiURL: "https://api-testnet.hecoinfo.com/api",
      browserURL: "https://testnet.hecoinfo.com",
    },
  },
  {
    network: "optimisticGoerli",
    chainId: 420,
    urls: {
      apiURL: "https://api-goerli-optimism.etherscan.io/api",
      browserURL: "https://goerli-optimism.etherscan.io/",
    },
  },
  {
    network: "polygonZkEVM",
    chainId: 1101,
    urls: {
      apiURL: "https://api-zkevm.polygonscan.com/api",
      browserURL: "https://zkevm.polygonscan.com",
    },
  },
  {
    network: "moonbeam",
    chainId: 1284,
    urls: {
      apiURL: "https://api-moonbeam.moonscan.io/api",
      browserURL: "https://moonbeam.moonscan.io",
    },
  },
  {
    network: "moonriver",
    chainId: 1285,
    urls: {
      apiURL: "https://api-moonriver.moonscan.io/api",
      browserURL: "https://moonriver.moonscan.io",
    },
  },
  {
    network: "moonbaseAlpha",
    chainId: 1287,
    urls: {
      apiURL: "https://api-moonbase.moonscan.io/api",
      browserURL: "https://moonbase.moonscan.io/",
    },
  },
  {
    network: "polygonZkEVMTestnet",
    chainId: 2442,
    urls: {
      apiURL: "https://api-cardona-zkevm.polygonscan.com/api",
      browserURL: "https://cardona-zkevm.polygonscan.com",
    },
  },
  {
    network: "ftmTestnet",
    chainId: 4002,
    urls: {
      apiURL: "https://api-testnet.ftmscan.com/api",
      browserURL: "https://testnet.ftmscan.com",
    },
  },
  {
    network: "base",
    chainId: 8453,
    urls: {
      apiURL: "https://api.basescan.org/api",
      browserURL: "https://basescan.org/",
    },
  },
  {
    network: "chiado",
    chainId: 10200,
    urls: {
      apiURL: "https://gnosis-chiado.blockscout.com/api",
      browserURL: "https://gnosis-chiado.blockscout.com",
    },
  },
  {
    network: "holesky",
    chainId: 17000,
    urls: {
      apiURL: "https://api-holesky.etherscan.io/api",
      browserURL: "https://holesky.etherscan.io",
    },
  },
  {
    network: "arbitrumOne",
    chainId: 42161,
    urls: {
      apiURL: "https://api.arbiscan.io/api",
      browserURL: "https://arbiscan.io/",
    },
  },
  {
    network: "avalancheFujiTestnet",
    chainId: 43113,
    urls: {
      apiURL: "https://api-testnet.snowtrace.io/api",
      browserURL: "https://testnet.snowtrace.io/",
    },
  },
  {
    network: "avalanche",
    chainId: 43114,
    urls: {
      apiURL: "https://api.snowtrace.io/api",
      browserURL: "https://snowtrace.io/",
    },
  },
  {
    network: "polygonMumbai",
    chainId: 80001,
    urls: {
      apiURL: "https://api-testnet.polygonscan.com/api",
      browserURL: "https://mumbai.polygonscan.com/",
    },
  },
  {
    network: "polygonAmoy",
    chainId: 80002,
    urls: {
      apiURL: "https://api-amoy.polygonscan.com/api",
      browserURL: "https://amoy.polygonscan.com/",
    },
  },
  {
    network: "baseGoerli",
    chainId: 84531,
    urls: {
      apiURL: "https://api-goerli.basescan.org/api",
      browserURL: "https://goerli.basescan.org/",
    },
  },
  {
    network: "baseSepolia",
    chainId: 84532,
    urls: {
      apiURL: "https://api-sepolia.basescan.org/api",
      browserURL: "https://sepolia.basescan.org/",
    },
  },
  {
    network: "arbitrumSepolia",
    chainId: 421614,
    urls: {
      apiURL: "https://api-sepolia.arbiscan.io/api",
      browserURL: "https://sepolia.arbiscan.io/",
    },
  },
  {
    network: "sepolia",
    chainId: 11155111,
    urls: {
      apiURL: "https://api-sepolia.etherscan.io/api",
      browserURL: "https://sepolia.etherscan.io",
    },
  },
  {
    network: "aurora",
    chainId: 1313161554,
    urls: {
      apiURL: "https://explorer.mainnet.aurora.dev/api",
      browserURL: "https://explorer.mainnet.aurora.dev",
    },
  },
  {
    network: "auroraTestnet",
    chainId: 1313161555,
    urls: {
      apiURL: "https://explorer.testnet.aurora.dev/api",
      browserURL: "https://explorer.testnet.aurora.dev",
    },
  },
  {
    network: "harmony",
    chainId: 1666600000,
    urls: {
      apiURL: "https://ctrver.t.hmny.io/verify",
      browserURL: "https://explorer.harmony.one",
    },
  },
  {
    network: "harmonyTest",
    chainId: 1666700000,
    urls: {
      apiURL: "https://ctrver.t.hmny.io/verify?network=testnet",
      browserURL: "https://explorer.pops.one",
    },
  },
  // We are not adding new networks to the core of hardhat-verify anymore.
  // Please read this to learn how to manually add support for custom networks:
  // https://github.com/NomicFoundation/hardhat/tree/main/packages/hardhat-verify#adding-support-for-other-networks
];
