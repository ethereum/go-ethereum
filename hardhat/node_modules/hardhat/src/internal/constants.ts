export const HARDHAT_NAME = "Hardhat";

export const HARDHAT_EXECUTABLE_NAME = "hardhat";
export const HARDHAT_NETWORK_NAME = "hardhat";

export const SOLIDITY_FILES_CACHE_FILENAME = "solidity-files-cache.json";

export const HARDHAT_NETWORK_SUPPORTED_HARDFORKS = [
  // "chainstart",
  // "homestead",
  // "dao",
  // "tangerineWhistle",
  // "spuriousDragon",
  "byzantium",
  "constantinople",
  "petersburg",
  "istanbul",
  "muirGlacier",
  "berlin",
  "london",
  "arrowGlacier",
  "grayGlacier",
  "merge",
  "shanghai",
  "cancun",
  "prague",
];

export const HARDHAT_MEMPOOL_SUPPORTED_ORDERS = ["fifo", "priority"] as const;

export const ARTIFACT_FORMAT_VERSION = "hh-sol-artifact-1";
export const DEBUG_FILE_FORMAT_VERSION = "hh-sol-dbg-1";
export const BUILD_INFO_FORMAT_VERSION = "hh-sol-build-info-1";
export const BUILD_INFO_DIR_NAME = "build-info";
export const EDIT_DISTANCE_THRESHOLD = 3;

export const HARDHAT_NETWORK_RESET_EVENT = "hardhatNetworkReset";
export const HARDHAT_NETWORK_REVERT_SNAPSHOT_EVENT =
  "hardhatNetworkRevertSnapshot";
