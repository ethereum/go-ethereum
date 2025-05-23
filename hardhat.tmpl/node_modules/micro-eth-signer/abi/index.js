"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.decodeEvent = exports.decodeTx = exports.decodeData = exports.tokenFromSymbol = exports.CONTRACTS = exports.TOKENS = exports.events = exports.deployContract = exports.createContract = exports.Decoder = exports.WETH = exports.UNISWAP_V3_ROUTER_CONTRACT = exports.UNISWAP_V2_ROUTER_CONTRACT = exports.KYBER_NETWORK_PROXY_CONTRACT = exports.ERC721 = exports.ERC20 = exports.ERC1155 = void 0;
const address_ts_1 = require("../address.js");
const index_ts_1 = require("../index.js");
const utils_ts_1 = require("../utils.js");
const decoder_ts_1 = require("./decoder.js");
Object.defineProperty(exports, "Decoder", { enumerable: true, get: function () { return decoder_ts_1.Decoder; } });
Object.defineProperty(exports, "createContract", { enumerable: true, get: function () { return decoder_ts_1.createContract; } });
Object.defineProperty(exports, "deployContract", { enumerable: true, get: function () { return decoder_ts_1.deployContract; } });
Object.defineProperty(exports, "events", { enumerable: true, get: function () { return decoder_ts_1.events; } });
const erc1155_ts_1 = require("./erc1155.js");
Object.defineProperty(exports, "ERC1155", { enumerable: true, get: function () { return erc1155_ts_1.default; } });
const erc20_ts_1 = require("./erc20.js");
Object.defineProperty(exports, "ERC20", { enumerable: true, get: function () { return erc20_ts_1.default; } });
const erc721_ts_1 = require("./erc721.js");
Object.defineProperty(exports, "ERC721", { enumerable: true, get: function () { return erc721_ts_1.default; } });
const kyber_ts_1 = require("./kyber.js");
Object.defineProperty(exports, "KYBER_NETWORK_PROXY_CONTRACT", { enumerable: true, get: function () { return kyber_ts_1.KYBER_NETWORK_PROXY_CONTRACT; } });
const uniswap_v2_ts_1 = require("./uniswap-v2.js");
Object.defineProperty(exports, "UNISWAP_V2_ROUTER_CONTRACT", { enumerable: true, get: function () { return uniswap_v2_ts_1.UNISWAP_V2_ROUTER_CONTRACT; } });
const uniswap_v3_ts_1 = require("./uniswap-v3.js");
Object.defineProperty(exports, "UNISWAP_V3_ROUTER_CONTRACT", { enumerable: true, get: function () { return uniswap_v3_ts_1.UNISWAP_V3_ROUTER_CONTRACT; } });
const weth_ts_1 = require("./weth.js");
Object.defineProperty(exports, "WETH", { enumerable: true, get: function () { return weth_ts_1.default; } });
exports.TOKENS = (() => Object.freeze(Object.fromEntries([
    ['UNI', '0x1f9840a85d5af5bf1d1762f925bdaddc4201f984'],
    ['BAT', '0x0d8775f648430679a709e98d2b0cb6250d2887ef'],
    // Required for Uniswap multi-hop routing
    ['USDT', '0xdac17f958d2ee523a2206206994597c13d831ec7', 6, 1],
    ['USDC', '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48', 6, 1],
    ['WETH', '0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2'],
    ['WBTC', '0x2260fac5e5542a773aa44fbcfedf7c193bc2c599', 8],
    ['DAI', '0x6b175474e89094c44da98b954eedeac495271d0f', 18, 1],
    ['COMP', '0xc00e94cb662c3520282e6f5717214004a7f26888'],
    ['MKR', '0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2'],
    ['AMPL', '0xd46ba6d942050d489dbd938a2c909a5d5039a161', 9],
].map(([symbol, addr, decimals, price]) => [
    addr,
    { abi: 'ERC20', symbol, decimals: decimals || 18, price },
]))))();
// <address, contractInfo>
exports.CONTRACTS = (() => Object.freeze({
    [uniswap_v2_ts_1.UNISWAP_V2_ROUTER_CONTRACT]: { abi: uniswap_v2_ts_1.default, name: 'UNISWAP V2 ROUTER' },
    [kyber_ts_1.KYBER_NETWORK_PROXY_CONTRACT]: { abi: kyber_ts_1.default, name: 'KYBER NETWORK PROXY' },
    [uniswap_v3_ts_1.UNISWAP_V3_ROUTER_CONTRACT]: { abi: uniswap_v3_ts_1.default, name: 'UNISWAP V3 ROUTER' },
    ...exports.TOKENS,
    [weth_ts_1.WETH_CONTRACT]: { abi: weth_ts_1.default, name: 'WETH Token', decimals: 18, symbol: 'WETH' },
}))();
const tokenFromSymbol = (symbol) => {
    for (let c in exports.TOKENS) {
        if (exports.TOKENS[c].symbol === symbol)
            return Object.assign({ contract: c }, exports.TOKENS[c]);
    }
    throw new Error('unknown token');
};
exports.tokenFromSymbol = tokenFromSymbol;
const getABI = (info) => {
    if (typeof info.abi === 'string') {
        if (info.abi === 'ERC20')
            return erc20_ts_1.default;
        else if (info.abi === 'ERC721')
            return erc721_ts_1.default;
        else
            throw new Error(`getABI: unknown abi type=${info.abi}`);
    }
    return info.abi;
};
// TODO: export? Seems useful enough
// We cannot have this inside decoder itself,
// since it will create dependencies on all default contracts
const getDecoder = (opt = {}) => {
    const decoder = new decoder_ts_1.Decoder();
    const contracts = {};
    // Add contracts
    if (!opt.noDefault)
        Object.assign(contracts, exports.CONTRACTS);
    if (opt.customContracts) {
        for (const k in opt.customContracts)
            contracts[k.toLowerCase()] = opt.customContracts[k];
    }
    // Contract info validation
    for (const k in contracts) {
        if (!address_ts_1.addr.isValid(k))
            throw new Error(`getDecoder: invalid contract address=${k}`);
        const c = contracts[k];
        if (c.symbol !== undefined && typeof c.symbol !== 'string')
            throw new Error(`getDecoder: wrong symbol type=${c.symbol}`);
        if (c.decimals !== undefined && !Number.isSafeInteger(c.decimals))
            throw new Error(`getDecoder: wrong decimals type=${c.decimals}`);
        if (c.name !== undefined && typeof c.name !== 'string')
            throw new Error(`getDecoder: wrong name type=${c.name}`);
        if (c.price !== undefined && typeof c.price !== 'number')
            throw new Error(`getDecoder: wrong price type=${c.price}`);
        decoder.add(k, getABI(c)); // validates c.abi
    }
    return { decoder, contracts };
};
// These methods are for case when user wants to inspect tx/logs/receipt,
// but doesn't know anything about which contract is used. If you work with
// specific contract it is better to use 'createContract' which will return nice types.
// 'to' can point to specific known contract, but also can point to any address (it is part of tx)
// 'to' should be part of real tx you want to parse, not hardcoded contract!
// Even if contract is unknown, we still try to process by known function signatures
// from other contracts.
// Can be used to parse tx or 'eth_getTransactionReceipt' output
const decodeData = (to, data, amount, opt = {}) => {
    if (!address_ts_1.addr.isValid(to))
        throw new Error(`decodeData: wrong to=${to}`);
    if (amount !== undefined && typeof amount !== 'bigint')
        throw new Error(`decodeData: wrong amount=${amount}`);
    const { decoder, contracts } = getDecoder(opt);
    return decoder.decode(to, utils_ts_1.ethHex.decode(data), {
        contract: to,
        contracts, // NOTE: we need whole contracts list here, since exchanges can use info about other contracts (tokens)
        contractInfo: contracts[to.toLowerCase()], // current contract info (for tokens)
        amount, // Amount is not neccesary, but some hints won't work without it (exchange eth to some tokens)
    });
};
exports.decodeData = decodeData;
// Requires deps on tx, but nicer API.
// Doesn't cover all use cases of decodeData, since it can't parse 'eth_getTransactionReceipt'
const decodeTx = (transaction, opt = {}) => {
    const tx = index_ts_1.Transaction.fromHex(transaction);
    return (0, exports.decodeData)(tx.raw.to, tx.raw.data, tx.raw.value, opt);
};
exports.decodeTx = decodeTx;
// Parses output of eth_getLogs/eth_getTransactionReceipt
const decodeEvent = (to, topics, data, opt = {}) => {
    if (!address_ts_1.addr.isValid(to))
        throw new Error(`decodeEvent: wrong to=${to}`);
    const { decoder, contracts } = getDecoder(opt);
    return decoder.decodeEvent(to, topics, data, {
        contract: to,
        contracts,
        contractInfo: contracts[to.toLowerCase()],
        // amount here is not used by our hooks. Should we ask it for consistency?
    });
};
exports.decodeEvent = decodeEvent;
//# sourceMappingURL=index.js.map