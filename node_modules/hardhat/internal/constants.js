"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.HARDHAT_NETWORK_REVERT_SNAPSHOT_EVENT = exports.HARDHAT_NETWORK_RESET_EVENT = exports.EDIT_DISTANCE_THRESHOLD = exports.BUILD_INFO_DIR_NAME = exports.BUILD_INFO_FORMAT_VERSION = exports.DEBUG_FILE_FORMAT_VERSION = exports.ARTIFACT_FORMAT_VERSION = exports.HARDHAT_MEMPOOL_SUPPORTED_ORDERS = exports.HARDHAT_NETWORK_SUPPORTED_HARDFORKS = exports.SOLIDITY_FILES_CACHE_FILENAME = exports.HARDHAT_NETWORK_NAME = exports.HARDHAT_EXECUTABLE_NAME = exports.HARDHAT_NAME = void 0;
exports.HARDHAT_NAME = "Hardhat";
exports.HARDHAT_EXECUTABLE_NAME = "hardhat";
exports.HARDHAT_NETWORK_NAME = "hardhat";
exports.SOLIDITY_FILES_CACHE_FILENAME = "solidity-files-cache.json";
exports.HARDHAT_NETWORK_SUPPORTED_HARDFORKS = [
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
];
exports.HARDHAT_MEMPOOL_SUPPORTED_ORDERS = ["fifo", "priority"];
exports.ARTIFACT_FORMAT_VERSION = "hh-sol-artifact-1";
exports.DEBUG_FILE_FORMAT_VERSION = "hh-sol-dbg-1";
exports.BUILD_INFO_FORMAT_VERSION = "hh-sol-build-info-1";
exports.BUILD_INFO_DIR_NAME = "build-info";
exports.EDIT_DISTANCE_THRESHOLD = 3;
exports.HARDHAT_NETWORK_RESET_EVENT = "hardhatNetworkReset";
exports.HARDHAT_NETWORK_REVERT_SNAPSHOT_EVENT = "hardhatNetworkRevertSnapshot";
//# sourceMappingURL=constants.js.map