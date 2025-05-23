declare const _default: {
    name: string;
    chainId: number;
    networkId: number;
    defaultHardfork: string;
    consensus: {
        type: string;
        algorithm: string;
        ethash: {};
    };
    comment: string;
    url: string;
    genesis: {
        timestamp: string;
        gasLimit: number;
        difficulty: number;
        nonce: string;
        extraData: string;
    };
    hardforks: ({
        name: string;
        block: number;
        forkHash: string;
        '//_comment'?: undefined;
        ttd?: undefined;
        timestamp?: undefined;
    } | {
        '//_comment': string;
        name: string;
        ttd: string;
        block: number;
        forkHash: string;
        timestamp?: undefined;
    } | {
        name: string;
        block: null;
        timestamp: string;
        forkHash: string;
        '//_comment'?: undefined;
        ttd?: undefined;
    })[];
    bootstrapNodes: never[];
    dnsNetworks: string[];
};
export default _default;
//# sourceMappingURL=sepolia.d.ts.map