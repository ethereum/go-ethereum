declare type rpcParams = {
    method: string;
    params: (string | boolean | number)[];
};
/**
 * Makes a simple RPC call to a remote Ethereum JSON-RPC provider and passes through the response.
 * No parameter or response validation is done.
 *
 * @param url the URL for the JSON RPC provider
 * @param params the parameters for the JSON-RPC method - refer to
 * https://ethereum.org/en/developers/docs/apis/json-rpc/ for details on RPC methods
 * @returns the `result` field from the JSON-RPC response
 * @example
 * ```ts
 * const provider = 'https://mainnet.infura.io/v3/...'
 * const params = {
 *   method: 'eth_getBlockByNumber',
 *   params: ['latest', false],
 * }
 *  const block = await fetchFromProvider(provider, params)
 */
export declare const fetchFromProvider: (url: string, params: rpcParams) => Promise<any>;
/**
 *
 * @param provider a URL string or {@link EthersProvider}
 * @returns the extracted URL string for the JSON-RPC Provider
 */
export declare const getProvider: (provider: string | EthersProvider) => string;
/**
 * A partial interface for an `ethers` `JsonRpcProvider`
 * We only use the url string since we do raw `fetch` calls to
 * retrieve the necessary data
 */
export interface EthersProvider {
    _getConnection: () => {
        url: string;
    };
}
export {};
//# sourceMappingURL=provider.d.ts.map