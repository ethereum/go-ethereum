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

import { EthExecutionAPI, Web3APISpec } from 'web3-types';
import { HttpProviderOptions } from 'web3-providers-http';
import { Network, SocketOptions, Transport } from './types.js';
import { Web3ExternalProvider } from './web3_provider.js';

const isValid = (str: string) => str !== undefined && str.trim().length > 0;

const websocketExclusions = [
	Network.DYMENSION_MAINNET,
	Network.DYMENSION_TESTNET,
	Network.KAVA_MAINNET,
	Network.CRONOS_MAINNET,
	// deprecated
	Network.POLYGON_MAINNET,
];

export class PublicNodeProvider<
	API extends Web3APISpec = EthExecutionAPI,
> extends Web3ExternalProvider<API> {
	// eslint-disable-next-line default-param-last
	public constructor(
		network: Network = Network.ETH_MAINNET,
		transport: Transport = Transport.HTTPS,
		host = '',
		providerConfigOptions?: HttpProviderOptions | SocketOptions,
	) {
		super(network, transport, '', host, providerConfigOptions);
	}
	public static readonly networkHostMap: { [key: string]: string } = {
		[Network.POLYGON_AMOY]: 'polygon-amoy-bor-rpc',
		[Network.DYMENSION_MAINNET]: 'dymension-evm-rpc',
		[Network.DYMENSION_TESTNET]: 'dymension-testnet-evm-rpc',
		[Network.BLAST_MAINNET]: 'blast-rpc',
		[Network.GNOSIS_MAINNET]: 'gnosis-rpc',
		[Network.PULSECHAIN_MAINNET]: 'pulsechain-rpc',
		[Network.PULSECHAIN_TESTNET]: 'pulsechain-testnet-rpc',
		[Network.KAVA_MAINNET]: 'kava-evm-rpc',
		[Network.CRONOS_MAINNET]: 'cronos-evm-rpc',
		[Network.MANTLE_MAINNET]: 'mantle-rpc',
		[Network.TAIKO_MAINNET]: 'taiko-rpc',
		[Network.TAIKO_HEKLA]: 'taiko-hekla-rpc',
		[Network.LINEA_MAINNET]: 'linea-rpc',
		[Network.LINEA_SEPOLIA]: 'linea-sepolia-rpc',
		[Network.SCROLL_MAINNET]: 'scroll-rpc',
		[Network.SCROLL_SEPOLIA]: 'scroll-sepolia-rpc',
		[Network.SYSCOIN_MAINNET]: 'syscoin-evm-rpc',
		[Network.SYSCOIN_TANENBAUM]: 'syscoin-tanenbaum-evm-rpc',
		[Network.HAQQ_MAINNET]: 'haqq-evm-rpc',
		[Network.EVMOS_MAINNET]: 'evmos-evm-rpc',
		[Network.EVMOS_TESTNET]: 'evmos-testnet-evm-rpc',
		[Network.BERACHAIN_TESTNET]: 'berachain-testnet-evm-rpc',
		[Network.ETH_MAINNET]: 'ethereum-rpc',
		[Network.ETH_SEPOLIA]: 'ethereum-sepolia-rpc',
		[Network.ETH_HOLESKY]: 'ethereum-holesky-rpc',
		[Network.BSC_MAINNET]: 'bsc-rpc',
		[Network.BSC_TESTNET]: 'bsc-testnet-rpc',
		[Network.POLYGON_MAINNET]: 'polygon-bor-rpc',
		[Network.BASE_MAINNET]: 'base-rpc',
		[Network.BASE_SEPOLIA]: 'base-sepolia-rpc',
		[Network.ARBITRUM_ONE]: 'arbitrum-one-rpc',
		[Network.ARBITRUM_NOVA]: 'arbitrum-nova-rpc',
		[Network.ARBITRUM_SEPOLIA]: 'arbitrum-sepolia-rpc',
		[Network.AVALANCHE_C_MAINNET]: 'avalanche-c-chain-rpc',
		[Network.AVALANCHE_P_MAINNET]: 'avalanche-p-chain-rpc',
		[Network.AVALANCHE_X_MAINNET]: 'avalanche-x-chain-rpc',
		[Network.AVALANCHE_FUJI_C]: 'avalanche-fuji-c-chain-rpc',
		[Network.AVALANCHE_FUJI_P]: 'avalanche-fuji-p-chain-rpc',
		[Network.AVALANCHE_FUJI_X]: 'avalanche-fuji-x-chain-rpc',
		[Network.OPTIMISM_MAINNET]: 'optimism-rpc',
		[Network.OPTIMISM_SEPOLIA]: 'optimism-sepolia-rpc',
		[Network.FANTOM_MAINNET]: 'fantom-rpc',
		[Network.FANTOM_TESTNET]: 'fantom-testnet-rpc',
		[Network.OPBNB_MAINNET]: 'opbnb-rpc',
		[Network.OPBNB_TESTNET]: 'opbnb-testnet-rpc',
		[Network.GNOSIS_CHIADO]: 'gnosis-chiado-rpc',
		[Network.CHILIZ_MAINNET]: 'chiliz-rpc',
		[Network.CHILIZ_SPICY]: 'chiliz-spicy-rpc',
		[Network.MOONBEAM_MAINNET]: 'moonbeam-rpc',
		[Network.BAHAMUT_MAINNET]: 'bahamut-rpc',
		[Network.TRON_MAINNET]: 'tron-evm-rpc',
		[Network.MOONRIVER_MAINNET]: 'moonriver-rpc',
	};
	// eslint-disable-next-line class-methods-use-this
	public getRPCURL(network: Network, transport: Transport, _: string, _host: string) {
		if (!PublicNodeProvider.networkHostMap[network]) {
			throw new Error('Network info not avalible.');
		}
		const defaultHost = `${PublicNodeProvider.networkHostMap[network]}.publicnode.com`;
		const host = isValid(_host) ? _host : defaultHost;
		if (websocketExclusions.includes(network) && transport === Transport.WebSocket) {
			return `${transport}://${host}/websocket`;
		}
		return `${transport}://${host}`;
	}
}
