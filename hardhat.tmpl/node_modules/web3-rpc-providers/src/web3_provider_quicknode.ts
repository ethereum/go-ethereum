/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/

import {
	EthExecutionAPI,
	JsonRpcResponseWithResult,
	Web3APIMethod,
	Web3APIPayload,
	Web3APIReturnType,
	Web3APISpec,
} from 'web3-types';
import { ResponseError } from 'web3-errors';
import { HttpProviderOptions } from 'web3-providers-http';
import { Transport, Network, SocketOptions } from './types.js';
import { Web3ExternalProvider } from './web3_provider.js';
import { QuickNodeRateLimitError } from './errors.js';

const isValid = (str: string) => str !== undefined && str.trim().length > 0;

export class QuickNodeProvider<
	API extends Web3APISpec = EthExecutionAPI,
> extends Web3ExternalProvider {
	// eslint-disable-next-line default-param-last
	public constructor(
		network: Network = Network.ETH_MAINNET,
		transport: Transport = Transport.HTTPS,
		token = '',
		host = '',
		providerConfigOptions?: HttpProviderOptions | SocketOptions,
	) {
		super(network, transport, token, host, providerConfigOptions);
	}

	public async request<
		Method extends Web3APIMethod<API>,
		ResultType = Web3APIReturnType<API, Method>,
	>(
		payload: Web3APIPayload<EthExecutionAPI, Method>,
		requestOptions?: RequestInit,
	): Promise<JsonRpcResponseWithResult<ResultType>> {
		try {
			return await super.request(payload, requestOptions);
		} catch (error) {
			if (error instanceof ResponseError && error.statusCode === 429) {
				throw new QuickNodeRateLimitError(error);
			}
			throw error;
		}
	}

	// eslint-disable-next-line class-methods-use-this
	public getRPCURL(network: Network, transport: Transport, _token: string, _host: string) {
		let host = '';
		let token = '';

		switch (network) {
			case Network.ETH_MAINNET:
				host = isValid(_host) ? _host : 'powerful-holy-bush.quiknode.pro';
				token = isValid(_token) ? _token : '3240624a343867035925ff7561eb60dfdba2a668';
				break;
			case Network.ETH_SEPOLIA:
				host = isValid(_host)
					? _host
					: 'dimensional-fabled-glitter.ethereum-sepolia.quiknode.pro';
				token = isValid(_token) ? _token : '382a3b5a4b938f2d6e8686c19af4b22921fde2cd';
				break;
			case Network.ETH_HOLESKY:
				host = isValid(_host) ? _host : 'yolo-morning-card.ethereum-holesky.quiknode.pro';
				token = isValid(_token) ? _token : '481ebe70638c4dcf176af617a16d02ab866b9af9';
				break;

			case Network.ARBITRUM_MAINNET:
				host = isValid(_host)
					? _host
					: 'autumn-divine-dinghy.arbitrum-mainnet.quiknode.pro';
				token = isValid(_token) ? _token : 'a5d7bfbf60b5ae9ce3628e53d69ef50d529e9a8c';
				break;
			case Network.ARBITRUM_SEPOLIA:
				host = isValid(_host) ? _host : 'few-patient-pond.arbitrum-sepolia.quiknode.pro';
				token = isValid(_token) ? _token : '3be985450970628c860b959c65cd2642dcafe53c';
				break;

			case Network.BNB_MAINNET:
				host = isValid(_host) ? _host : 'purple-empty-reel.bsc.quiknode.pro';
				token = isValid(_token) ? _token : 'ebf6c532961e21f092ff2facce1ec4c89c540158';
				break;
			case Network.BNB_TESTNET:
				host = isValid(_host) ? _host : 'floral-rough-scion.bsc-testnet.quiknode.pro';
				token = isValid(_token) ? _token : '5b297e5acff5f81f4c37ebf6f235f7299b6f9d28';
				break;

			case Network.POLYGON_MAINNET:
				host = isValid(_host) ? _host : 'small-chaotic-moon.matic.quiknode.pro';
				token = isValid(_token) ? _token : '847569f8a017e84d985e10d0f44365d965a951f1';
				break;
			case Network.POLYGON_AMOY:
				host = isValid(_host) ? _host : 'prettiest-side-shape.matic-amoy.quiknode.pro';
				token = isValid(_token) ? _token : '79a9476eea661d4f82de614db1d8a895b14b881c';
				break;
			default:
				throw new Error('Network info not avalible.');
		}

		return `${transport}://${host}/${token}`;
	}
}
