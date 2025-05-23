"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getLastSafeBlockNumber = exports.makeForkClient = exports.makeForkProvider = void 0;
const picocolors_1 = __importDefault(require("picocolors"));
const constants_1 = require("../../../constants");
const errors_1 = require("../../../core/errors");
const base_types_1 = require("../../../core/jsonrpc/types/base-types");
const http_1 = require("../../../core/providers/http");
const client_1 = require("../../jsonrpc/client");
const reorgs_protection_1 = require("./reorgs-protection");
// TODO: This is a temporarily measure.
//  We must investigate why this timeouts so much. Apparently
//  node-fetch doesn't handle timeouts so well. The option was
//  removed in its new major version. UPDATE: we aren't even using node-fetch
//  anymore, so this really should be revisited.
const FORK_HTTP_TIMEOUT = 35000;
async function makeForkProvider(forkConfig) {
    const forkProvider = new http_1.HttpProvider(forkConfig.jsonRpcUrl, constants_1.HARDHAT_NETWORK_NAME, forkConfig.httpHeaders, FORK_HTTP_TIMEOUT);
    const networkId = await getNetworkId(forkProvider);
    const actualMaxReorg = (0, reorgs_protection_1.getLargestPossibleReorg)(networkId);
    const maxReorg = actualMaxReorg ?? reorgs_protection_1.FALLBACK_MAX_REORG;
    const latestBlockNumber = await getLatestBlockNumber(forkProvider);
    const lastSafeBlockNumber = getLastSafeBlockNumber(latestBlockNumber, maxReorg);
    let forkBlockNumber;
    if (forkConfig.blockNumber !== undefined) {
        if (forkConfig.blockNumber > latestBlockNumber) {
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw new Error(`Trying to initialize a provider with block ${forkConfig.blockNumber} but the current block is ${latestBlockNumber}`);
        }
        if (forkConfig.blockNumber > lastSafeBlockNumber) {
            const confirmations = latestBlockNumber - BigInt(forkConfig.blockNumber) + 1n;
            const requiredConfirmations = maxReorg + 1n;
            console.warn(picocolors_1.default.yellow(`You are forking from block ${forkConfig.blockNumber}, which has less than ${requiredConfirmations} confirmations, and will affect Hardhat Network's performance.
Please use block number ${lastSafeBlockNumber} or wait for the block to get ${requiredConfirmations - confirmations} more confirmations.`));
        }
        forkBlockNumber = BigInt(forkConfig.blockNumber);
    }
    else {
        forkBlockNumber = BigInt(lastSafeBlockNumber);
    }
    return {
        forkProvider,
        networkId,
        forkBlockNumber,
        latestBlockNumber,
        maxReorg,
    };
}
exports.makeForkProvider = makeForkProvider;
async function makeForkClient(forkConfig, forkCachePath) {
    const { forkProvider, networkId, forkBlockNumber, latestBlockNumber, maxReorg, } = await makeForkProvider(forkConfig);
    const block = await getBlockByNumber(forkProvider, forkBlockNumber);
    const forkBlockTimestamp = (0, base_types_1.rpcQuantityToNumber)(block.timestamp) * 1000;
    const cacheToDiskEnabled = forkConfig.blockNumber !== undefined && forkCachePath !== undefined;
    const forkClient = new client_1.JsonRpcClient(forkProvider, networkId, latestBlockNumber, maxReorg, cacheToDiskEnabled ? forkCachePath : undefined);
    const forkBlockHash = block.hash;
    (0, errors_1.assertHardhatInvariant)(forkBlockHash !== null, "Forked block should have a hash");
    const forkBlockStateRoot = block.stateRoot;
    return {
        forkClient,
        forkBlockNumber,
        forkBlockTimestamp,
        forkBlockHash,
        forkBlockStateRoot,
    };
}
exports.makeForkClient = makeForkClient;
async function getBlockByNumber(provider, blockNumber) {
    const rpcBlockOutput = (await provider.request({
        method: "eth_getBlockByNumber",
        params: [(0, base_types_1.numberToRpcQuantity)(blockNumber), false],
    }));
    return rpcBlockOutput;
}
async function getNetworkId(provider) {
    const networkIdString = (await provider.request({
        method: "net_version",
    }));
    return parseInt(networkIdString, 10);
}
async function getLatestBlockNumber(provider) {
    const latestBlockString = (await provider.request({
        method: "eth_blockNumber",
    }));
    const latestBlock = BigInt(latestBlockString);
    return latestBlock;
}
function getLastSafeBlockNumber(latestBlockNumber, maxReorg) {
    // Design choice: if latestBlock - maxReorg results in a negative number then the latestBlock block will be used.
    // This decision is based on the assumption that if maxReorg > latestBlock then there is a high probability that the fork is occurring on a devnet.
    return latestBlockNumber - maxReorg >= 0
        ? latestBlockNumber - maxReorg
        : latestBlockNumber;
}
exports.getLastSafeBlockNumber = getLastSafeBlockNumber;
//# sourceMappingURL=makeForkClient.js.map