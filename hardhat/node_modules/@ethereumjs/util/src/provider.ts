type rpcParams = {
  method: string
  params: (string | boolean | number)[]
}

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
 * const block = await fetchFromProvider(provider, params)
 * ```
 */
export const fetchFromProvider = async (url: string, params: rpcParams) => {
  const data = JSON.stringify({
    method: params.method,
    params: params.params,
    jsonrpc: '2.0',
    id: 1,
  })

  const res = await fetch(url, {
    headers: {
      'content-type': 'application/json',
    },
    method: 'POST',
    body: data,
  })
  if (!res.ok) {
    throw new Error(
      `JSONRpcError: ${JSON.stringify(
        {
          method: params.method,
          status: res.status,
          message: await res.text().catch(() => {
            return 'Could not parse error message likely because of a network error'
          }),
        },
        null,
        2
      )}`
    )
  }
  const json = await res.json()
  // TODO we should check json.error here
  return json.result
}

/**
 *
 * @param provider a URL string or {@link EthersProvider}
 * @returns the extracted URL string for the JSON-RPC Provider
 */
export const getProvider = (provider: string | EthersProvider) => {
  if (typeof provider === 'string') {
    return provider
  } else if (typeof provider === 'object' && provider._getConnection !== undefined) {
    return provider._getConnection().url
  } else {
    throw new Error('Must provide valid provider URL or Web3Provider')
  }
}

/**
 * A partial interface for an `ethers` `JsonRpcProvider`
 * We only use the url string since we do raw `fetch` calls to
 * retrieve the necessary data
 */
export interface EthersProvider {
  _getConnection: () => {
    url: string
  }
}
