declare const _default: {
    name: string;
    chainId: number;
    networkId: number;
    defaultHardfork: string;
    consensus: {
        type: string;
        algorithm: string;
        clique: {
            period: number;
            epoch: number;
        };
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
    } | {
        '//_comment': string;
        name: string;
        ttd: string;
        block: number;
        forkHash: string;
    } | {
        name: string;
        block: null;
        forkHash: null;
        '//_comment'?: undefined;
        ttd?: undefined;
    })[];
    bootstrapNodes: never[];
    dnsNetworks: string[];
};
export default _default;
//# sourceMappingURL=goerli.d.ts.map