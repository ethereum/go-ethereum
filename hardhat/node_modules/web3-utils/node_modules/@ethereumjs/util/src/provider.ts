import fetch from 'micro-ftch'

type rpcParams = {
  method: string
  params: (string | boolean | number)[]
}
export const fetchFromProvider = async (url: string, params: rpcParams) => {
  const res = await fetch(url, {
    headers: {
      'content-type': 'application/json',
    },
    type: 'json',
    data: {
      method: params.method,
      params: params.params,
      jsonrpc: '2.0',
      id: 1,
    },
  })

  return res.result
}

export const getProvider = (provider: string | any) => {
  if (typeof provider === 'string') {
    return provider
  } else if (provider?.connection?.url !== undefined) {
    return provider.connection.url
  } else {
    throw new Error('Must provide valid provider URL or Web3Provider')
  }
}
