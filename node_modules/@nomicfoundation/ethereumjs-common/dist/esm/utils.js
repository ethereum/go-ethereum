import { intToHex, isHexPrefixed, stripHexPrefix } from '@nomicfoundation/ethereumjs-util';
import { Hardfork } from './enums.js';
/**
 * Transforms Geth formatted nonce (i.e. hex string) to 8 byte 0x-prefixed string used internally
 * @param nonce string parsed from the Geth genesis file
 * @returns nonce as a 0x-prefixed 8 byte string
 */
function formatNonce(nonce) {
    if (!nonce || nonce === '0x0') {
        return '0x0000000000000000';
    }
    if (isHexPrefixed(nonce)) {
        return '0x' + stripHexPrefix(nonce).padStart(16, '0');
    }
    return '0x' + nonce.padStart(16, '0');
}
/**
 * Converts Geth genesis parameters to an EthereumJS compatible `CommonOpts` object
 * @param json object representing the Geth genesis file
 * @param optional mergeForkIdPostMerge which clarifies the placement of MergeForkIdTransition
 * hardfork, which by default is post merge as with the merged eth networks but could also come
 * before merge like in kiln genesis
 * @returns genesis parameters in a `CommonOpts` compliant object
 */
function parseGethParams(json, mergeForkIdPostMerge = true) {
    const { name, config, difficulty, mixHash, gasLimit, coinbase, baseFeePerGas, excessBlobGas, } = json;
    let { extraData, timestamp, nonce } = json;
    const genesisTimestamp = Number(timestamp);
    const { chainId } = config;
    // geth is not strictly putting empty fields with a 0x prefix
    if (extraData === '') {
        extraData = '0x';
    }
    // geth may use number for timestamp
    if (!isHexPrefixed(timestamp)) {
        timestamp = intToHex(parseInt(timestamp));
    }
    // geth may not give us a nonce strictly formatted to an 8 byte hex string
    if (nonce.length !== 18) {
        nonce = formatNonce(nonce);
    }
    // EIP155 and EIP158 are both part of Spurious Dragon hardfork and must occur at the same time
    // but have different configuration parameters in geth genesis parameters
    if (config.eip155Block !== config.eip158Block) {
        throw new Error('EIP155 block number must equal EIP 158 block number since both are part of SpuriousDragon hardfork and the client only supports activating the full hardfork');
    }
    const params = {
        name,
        chainId,
        networkId: chainId,
        genesis: {
            timestamp,
            gasLimit,
            difficulty,
            nonce,
            extraData,
            mixHash,
            coinbase,
            baseFeePerGas,
            excessBlobGas,
        },
        hardfork: undefined,
        hardforks: [],
        bootstrapNodes: [],
        consensus: config.clique !== undefined
            ? {
                type: 'poa',
                algorithm: 'clique',
                clique: {
                    // The recent geth genesis seems to be using blockperiodseconds
                    // and epochlength for clique specification
                    // see: https://hackmd.io/PqZgMpnkSWCWv5joJoFymQ
                    period: config.clique.period ?? config.clique.blockperiodseconds,
                    epoch: config.clique.epoch ?? config.clique.epochlength,
                },
            }
            : {
                type: 'pow',
                algorithm: 'ethash',
                ethash: {},
            },
    };
    const forkMap = {
        [Hardfork.Homestead]: { name: 'homesteadBlock' },
        [Hardfork.Dao]: { name: 'daoForkBlock' },
        [Hardfork.TangerineWhistle]: { name: 'eip150Block' },
        [Hardfork.SpuriousDragon]: { name: 'eip155Block' },
        [Hardfork.Byzantium]: { name: 'byzantiumBlock' },
        [Hardfork.Constantinople]: { name: 'constantinopleBlock' },
        [Hardfork.Petersburg]: { name: 'petersburgBlock' },
        [Hardfork.Istanbul]: { name: 'istanbulBlock' },
        [Hardfork.MuirGlacier]: { name: 'muirGlacierBlock' },
        [Hardfork.Berlin]: { name: 'berlinBlock' },
        [Hardfork.London]: { name: 'londonBlock' },
        [Hardfork.MergeForkIdTransition]: { name: 'mergeForkBlock', postMerge: mergeForkIdPostMerge },
        [Hardfork.Shanghai]: { name: 'shanghaiTime', postMerge: true, isTimestamp: true },
        [Hardfork.Cancun]: { name: 'cancunTime', postMerge: true, isTimestamp: true },
        [Hardfork.Prague]: { name: 'pragueTime', postMerge: true, isTimestamp: true },
    };
    // forkMapRev is the map from config field name to Hardfork
    const forkMapRev = Object.keys(forkMap).reduce((acc, elem) => {
        acc[forkMap[elem].name] = elem;
        return acc;
    }, {});
    const configHardforkNames = Object.keys(config).filter((key) => forkMapRev[key] !== undefined && config[key] !== undefined && config[key] !== null);
    params.hardforks = configHardforkNames
        .map((nameBlock) => ({
        name: forkMapRev[nameBlock],
        block: forkMap[forkMapRev[nameBlock]].isTimestamp === true || typeof config[nameBlock] !== 'number'
            ? null
            : config[nameBlock],
        timestamp: forkMap[forkMapRev[nameBlock]].isTimestamp === true && typeof config[nameBlock] === 'number'
            ? config[nameBlock]
            : undefined,
    }))
        .filter((fork) => fork.block !== null || fork.timestamp !== undefined);
    params.hardforks.sort(function (a, b) {
        return (a.block ?? Infinity) - (b.block ?? Infinity);
    });
    params.hardforks.sort(function (a, b) {
        // non timestamp forks come before any timestamp forks
        return (a.timestamp ?? 0) - (b.timestamp ?? 0);
    });
    // only set the genesis timestamp forks to zero post the above sort has happended
    // to get the correct sorting
    for (const hf of params.hardforks) {
        if (hf.timestamp === genesisTimestamp) {
            hf.timestamp = 0;
        }
    }
    if (config.terminalTotalDifficulty !== undefined) {
        // Following points need to be considered for placement of merge hf
        // - Merge hardfork can't be placed at genesis
        // - Place merge hf before any hardforks that require CL participation for e.g. withdrawals
        // - Merge hardfork has to be placed just after genesis if any of the genesis hardforks make CL
        //   necessary for e.g. withdrawals
        const mergeConfig = {
            name: Hardfork.Paris,
            ttd: config.terminalTotalDifficulty,
            block: null,
        };
        // Merge hardfork has to be placed before first hardfork that is dependent on merge
        const postMergeIndex = params.hardforks.findIndex((hf) => forkMap[hf.name]?.postMerge === true);
        if (postMergeIndex !== -1) {
            params.hardforks.splice(postMergeIndex, 0, mergeConfig);
        }
        else {
            params.hardforks.push(mergeConfig);
        }
    }
    const latestHardfork = params.hardforks.length > 0 ? params.hardforks.slice(-1)[0] : undefined;
    params.hardfork = latestHardfork?.name;
    params.hardforks.unshift({ name: Hardfork.Chainstart, block: 0 });
    return params;
}
/**
 * Parses a genesis.json exported from Geth into parameters for Common instance
 * @param json representing the Geth genesis file
 * @param name optional chain name
 * @returns parsed params
 */
export function parseGethGenesis(json, name, mergeForkIdPostMerge) {
    try {
        const required = ['config', 'difficulty', 'gasLimit', 'nonce', 'alloc'];
        if (required.some((field) => !(field in json))) {
            const missingField = required.filter((field) => !(field in json));
            throw new Error(`Invalid format, expected geth genesis field "${missingField}" missing`);
        }
        if (name !== undefined) {
            json.name = name;
        }
        return parseGethParams(json, mergeForkIdPostMerge);
    }
    catch (e) {
        throw new Error(`Error parsing parameters file: ${e.message}`);
    }
}
//# sourceMappingURL=utils.js.map