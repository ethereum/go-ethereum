export declare enum Chain {
    Mainnet = 1,
    Goerli = 5,
    Sepolia = 11155111
}
export declare enum Hardfork {
    Chainstart = "chainstart",
    Homestead = "homestead",
    Dao = "dao",
    TangerineWhistle = "tangerineWhistle",
    SpuriousDragon = "spuriousDragon",
    Byzantium = "byzantium",
    Constantinople = "constantinople",
    Petersburg = "petersburg",
    Istanbul = "istanbul",
    MuirGlacier = "muirGlacier",
    Berlin = "berlin",
    London = "london",
    ArrowGlacier = "arrowGlacier",
    GrayGlacier = "grayGlacier",
    MergeForkIdTransition = "mergeForkIdTransition",
    Merge = "merge",
    Shanghai = "shanghai",
    ShardingForkDev = "shardingFork"
}
export declare enum ConsensusType {
    ProofOfStake = "pos",
    ProofOfWork = "pow",
    ProofOfAuthority = "poa"
}
export declare enum ConsensusAlgorithm {
    Ethash = "ethash",
    Clique = "clique",
    Casper = "casper"
}
export declare enum CustomChain {
    /**
     * Polygon (Matic) Mainnet
     *
     * - [Documentation](https://docs.matic.network/docs/develop/network-details/network)
     */
    PolygonMainnet = "polygon-mainnet",
    /**
     * Polygon (Matic) Mumbai Testnet
     *
     * - [Documentation](https://docs.matic.network/docs/develop/network-details/network)
     */
    PolygonMumbai = "polygon-mumbai",
    /**
     * Arbitrum Rinkeby Testnet
     *
     * - [Documentation](https://developer.offchainlabs.com/docs/public_testnet)
     */
    ArbitrumRinkebyTestnet = "arbitrum-rinkeby-testnet",
    /**
     * Arbitrum One - mainnet for Arbitrum roll-up
     *
     * - [Documentation](https://developer.offchainlabs.com/public-chains)
     */
    ArbitrumOne = "arbitrum-one",
    /**
     * xDai EVM sidechain with a native stable token
     *
     * - [Documentation](https://www.xdaichain.com/)
     */
    xDaiChain = "x-dai-chain",
    /**
     * Optimistic Kovan - testnet for Optimism roll-up
     *
     * - [Documentation](https://community.optimism.io/docs/developers/tutorials.html)
     */
    OptimisticKovan = "optimistic-kovan",
    /**
     * Optimistic Ethereum - mainnet for Optimism roll-up
     *
     * - [Documentation](https://community.optimism.io/docs/developers/tutorials.html)
     */
    OptimisticEthereum = "optimistic-ethereum"
}
//# sourceMappingURL=enums.d.ts.map