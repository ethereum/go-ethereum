"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.CustomChain = exports.ConsensusAlgorithm = exports.ConsensusType = exports.Hardfork = exports.ChainGenesis = exports.Chain = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
var Chain;
(function (Chain) {
    Chain[Chain["Mainnet"] = 1] = "Mainnet";
    Chain[Chain["Goerli"] = 5] = "Goerli";
    Chain[Chain["Sepolia"] = 11155111] = "Sepolia";
    Chain[Chain["Holesky"] = 17000] = "Holesky";
    Chain[Chain["Kaustinen"] = 69420] = "Kaustinen";
})(Chain = exports.Chain || (exports.Chain = {}));
// Having this info as record will force typescript to make sure no chain is missed
/**
 * GenesisState info about well known ethereum chains
 */
exports.ChainGenesis = {
    [Chain.Mainnet]: {
        name: 'mainnet',
        blockNumber: ethereumjs_util_1.BIGINT_0,
        stateRoot: (0, ethereumjs_util_1.hexToBytes)('0xd7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544'),
    },
    [Chain.Goerli]: {
        name: 'goerli',
        blockNumber: ethereumjs_util_1.BIGINT_0,
        stateRoot: (0, ethereumjs_util_1.hexToBytes)('0x5d6cded585e73c4e322c30c2f782a336316f17dd85a4863b9d838d2d4b8b3008'),
    },
    [Chain.Sepolia]: {
        name: 'sepolia',
        blockNumber: ethereumjs_util_1.BIGINT_0,
        stateRoot: (0, ethereumjs_util_1.hexToBytes)('0x5eb6e371a698b8d68f665192350ffcecbbbf322916f4b51bd79bb6887da3f494'),
    },
    [Chain.Holesky]: {
        name: 'holesky',
        blockNumber: ethereumjs_util_1.BIGINT_0,
        stateRoot: (0, ethereumjs_util_1.hexToBytes)('0x69d8c9d72f6fa4ad42d4702b433707212f90db395eb54dc20bc85de253788783'),
    },
    [Chain.Kaustinen]: {
        name: 'kaustinen',
        blockNumber: ethereumjs_util_1.BIGINT_0,
        stateRoot: (0, ethereumjs_util_1.hexToBytes)('0x5e8519756841faf0b2c28951c451b61a4b407b70a5ce5b57992f4bec973173ff'),
    },
};
var Hardfork;
(function (Hardfork) {
    Hardfork["Chainstart"] = "chainstart";
    Hardfork["Homestead"] = "homestead";
    Hardfork["Dao"] = "dao";
    Hardfork["TangerineWhistle"] = "tangerineWhistle";
    Hardfork["SpuriousDragon"] = "spuriousDragon";
    Hardfork["Byzantium"] = "byzantium";
    Hardfork["Constantinople"] = "constantinople";
    Hardfork["Petersburg"] = "petersburg";
    Hardfork["Istanbul"] = "istanbul";
    Hardfork["MuirGlacier"] = "muirGlacier";
    Hardfork["Berlin"] = "berlin";
    Hardfork["London"] = "london";
    Hardfork["ArrowGlacier"] = "arrowGlacier";
    Hardfork["GrayGlacier"] = "grayGlacier";
    Hardfork["MergeForkIdTransition"] = "mergeForkIdTransition";
    Hardfork["Paris"] = "paris";
    Hardfork["Shanghai"] = "shanghai";
    Hardfork["Cancun"] = "cancun";
    Hardfork["Prague"] = "prague";
})(Hardfork = exports.Hardfork || (exports.Hardfork = {}));
var ConsensusType;
(function (ConsensusType) {
    ConsensusType["ProofOfStake"] = "pos";
    ConsensusType["ProofOfWork"] = "pow";
    ConsensusType["ProofOfAuthority"] = "poa";
})(ConsensusType = exports.ConsensusType || (exports.ConsensusType = {}));
var ConsensusAlgorithm;
(function (ConsensusAlgorithm) {
    ConsensusAlgorithm["Ethash"] = "ethash";
    ConsensusAlgorithm["Clique"] = "clique";
    ConsensusAlgorithm["Casper"] = "casper";
})(ConsensusAlgorithm = exports.ConsensusAlgorithm || (exports.ConsensusAlgorithm = {}));
var CustomChain;
(function (CustomChain) {
    /**
     * Polygon (Matic) Mainnet
     *
     * - [Documentation](https://docs.matic.network/docs/develop/network-details/network)
     */
    CustomChain["PolygonMainnet"] = "polygon-mainnet";
    /**
     * Polygon (Matic) Mumbai Testnet
     *
     * - [Documentation](https://docs.matic.network/docs/develop/network-details/network)
     */
    CustomChain["PolygonMumbai"] = "polygon-mumbai";
    /**
     * Arbitrum One - mainnet for Arbitrum roll-up
     *
     * - [Documentation](https://developer.offchainlabs.com/public-chains)
     */
    CustomChain["ArbitrumOne"] = "arbitrum-one";
    /**
     * xDai EVM sidechain with a native stable token
     *
     * - [Documentation](https://www.xdaichain.com/)
     */
    CustomChain["xDaiChain"] = "x-dai-chain";
    /**
     * Optimistic Kovan - testnet for Optimism roll-up
     *
     * - [Documentation](https://community.optimism.io/docs/developers/tutorials.html)
     */
    CustomChain["OptimisticKovan"] = "optimistic-kovan";
    /**
     * Optimistic Ethereum - mainnet for Optimism roll-up
     *
     * - [Documentation](https://community.optimism.io/docs/developers/tutorials.html)
     */
    CustomChain["OptimisticEthereum"] = "optimistic-ethereum";
})(CustomChain = exports.CustomChain || (exports.CustomChain = {}));
//# sourceMappingURL=enums.js.map