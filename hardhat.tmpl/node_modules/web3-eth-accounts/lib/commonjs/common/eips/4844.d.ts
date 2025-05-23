declare const _default: {
    name: string;
    number: number;
    comment: string;
    url: string;
    status: string;
    minimumHardfork: string;
    requiredEIPs: number[];
    gasConfig: {
        dataGasPerBlob: {
            v: number;
            d: string;
        };
        targetDataGasPerBlock: {
            v: number;
            d: string;
        };
        maxDataGasPerBlock: {
            v: number;
            d: string;
        };
        dataGasPriceUpdateFraction: {
            v: number;
            d: string;
        };
    };
    gasPrices: {
        simpleGasPerBlob: {
            v: number;
            d: string;
        };
        minDataGasPrice: {
            v: number;
            d: string;
        };
        kzgPointEvaluationGasPrecompilePrice: {
            v: number;
            d: string;
        };
        datahash: {
            v: number;
            d: string;
        };
    };
    sharding: {
        blobCommitmentVersionKzg: {
            v: number;
            d: string;
        };
        fieldElementsPerBlob: {
            v: number;
            d: string;
        };
    };
    vm: {};
    pow: {};
};
export default _default;
