declare type rpcParams = {
    method: string;
    params: (string | boolean | number)[];
};
export declare const fetchFromProvider: (url: string, params: rpcParams) => Promise<any>;
export declare const getProvider: (provider: string | any) => any;
export {};
//# sourceMappingURL=provider.d.ts.map