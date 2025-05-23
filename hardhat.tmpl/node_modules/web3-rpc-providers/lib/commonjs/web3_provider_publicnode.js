"use strict";
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.PublicNodeProvider = void 0;
const types_js_1 = require("./types.js");
const web3_provider_js_1 = require("./web3_provider.js");
const isValid = (str) => str !== undefined && str.trim().length > 0;
const websocketExclusions = [
    types_js_1.Network.DYMENSION_MAINNET,
    types_js_1.Network.DYMENSION_TESTNET,
    types_js_1.Network.KAVA_MAINNET,
    types_js_1.Network.CRONOS_MAINNET,
    // deprecated
    types_js_1.Network.POLYGON_MAINNET,
];
class PublicNodeProvider extends web3_provider_js_1.Web3ExternalProvider {
    // eslint-disable-next-line default-param-last
    constructor(network = types_js_1.Network.ETH_MAINNET, transport = types_js_1.Transport.HTTPS, host = '', providerConfigOptions) {
        super(network, transport, '', host, providerConfigOptions);
    }
    // eslint-disable-next-line class-methods-use-this
    getRPCURL(network, transport, _, _host) {
        if (!PublicNodeProvider.networkHostMap[network]) {
            throw new Error('Network info not avalible.');
        }
        const defaultHost = `${PublicNodeProvider.networkHostMap[network]}.publicnode.com`;
        const host = isValid(_host) ? _host : defaultHost;
        if (websocketExclusions.includes(network) && transport === types_js_1.Transport.WebSocket) {
            return `${transport}://${host}/websocket`;
        }
        return `${transport}://${host}`;
    }
}
exports.PublicNodeProvider = PublicNodeProvider;
PublicNodeProvider.networkHostMap = {
    [types_js_1.Network.POLYGON_AMOY]: 'polygon-amoy-bor-rpc',
    [types_js_1.Network.DYMENSION_MAINNET]: 'dymension-evm-rpc',
    [types_js_1.Network.DYMENSION_TESTNET]: 'dymension-testnet-evm-rpc',
    [types_js_1.Network.BLAST_MAINNET]: 'blast-rpc',
    [types_js_1.Network.GNOSIS_MAINNET]: 'gnosis-rpc',
    [types_js_1.Network.PULSECHAIN_MAINNET]: 'pulsechain-rpc',
    [types_js_1.Network.PULSECHAIN_TESTNET]: 'pulsechain-testnet-rpc',
    [types_js_1.Network.KAVA_MAINNET]: 'kava-evm-rpc',
    [types_js_1.Network.CRONOS_MAINNET]: 'cronos-evm-rpc',
    [types_js_1.Network.MANTLE_MAINNET]: 'mantle-rpc',
    [types_js_1.Network.TAIKO_MAINNET]: 'taiko-rpc',
    [types_js_1.Network.TAIKO_HEKLA]: 'taiko-hekla-rpc',
    [types_js_1.Network.LINEA_MAINNET]: 'linea-rpc',
    [types_js_1.Network.LINEA_SEPOLIA]: 'linea-sepolia-rpc',
    [types_js_1.Network.SCROLL_MAINNET]: 'scroll-rpc',
    [types_js_1.Network.SCROLL_SEPOLIA]: 'scroll-sepolia-rpc',
    [types_js_1.Network.SYSCOIN_MAINNET]: 'syscoin-evm-rpc',
    [types_js_1.Network.SYSCOIN_TANENBAUM]: 'syscoin-tanenbaum-evm-rpc',
    [types_js_1.Network.HAQQ_MAINNET]: 'haqq-evm-rpc',
    [types_js_1.Network.EVMOS_MAINNET]: 'evmos-evm-rpc',
    [types_js_1.Network.EVMOS_TESTNET]: 'evmos-testnet-evm-rpc',
    [types_js_1.Network.BERACHAIN_TESTNET]: 'berachain-testnet-evm-rpc',
    [types_js_1.Network.ETH_MAINNET]: 'ethereum-rpc',
    [types_js_1.Network.ETH_SEPOLIA]: 'ethereum-sepolia-rpc',
    [types_js_1.Network.ETH_HOLESKY]: 'ethereum-holesky-rpc',
    [types_js_1.Network.BSC_MAINNET]: 'bsc-rpc',
    [types_js_1.Network.BSC_TESTNET]: 'bsc-testnet-rpc',
    [types_js_1.Network.POLYGON_MAINNET]: 'polygon-bor-rpc',
    [types_js_1.Network.BASE_MAINNET]: 'base-rpc',
    [types_js_1.Network.BASE_SEPOLIA]: 'base-sepolia-rpc',
    [types_js_1.Network.ARBITRUM_ONE]: 'arbitrum-one-rpc',
    [types_js_1.Network.ARBITRUM_NOVA]: 'arbitrum-nova-rpc',
    [types_js_1.Network.ARBITRUM_SEPOLIA]: 'arbitrum-sepolia-rpc',
    [types_js_1.Network.AVALANCHE_C_MAINNET]: 'avalanche-c-chain-rpc',
    [types_js_1.Network.AVALANCHE_P_MAINNET]: 'avalanche-p-chain-rpc',
    [types_js_1.Network.AVALANCHE_X_MAINNET]: 'avalanche-x-chain-rpc',
    [types_js_1.Network.AVALANCHE_FUJI_C]: 'avalanche-fuji-c-chain-rpc',
    [types_js_1.Network.AVALANCHE_FUJI_P]: 'avalanche-fuji-p-chain-rpc',
    [types_js_1.Network.AVALANCHE_FUJI_X]: 'avalanche-fuji-x-chain-rpc',
    [types_js_1.Network.OPTIMISM_MAINNET]: 'optimism-rpc',
    [types_js_1.Network.OPTIMISM_SEPOLIA]: 'optimism-sepolia-rpc',
    [types_js_1.Network.FANTOM_MAINNET]: 'fantom-rpc',
    [types_js_1.Network.FANTOM_TESTNET]: 'fantom-testnet-rpc',
    [types_js_1.Network.OPBNB_MAINNET]: 'opbnb-rpc',
    [types_js_1.Network.OPBNB_TESTNET]: 'opbnb-testnet-rpc',
    [types_js_1.Network.GNOSIS_CHIADO]: 'gnosis-chiado-rpc',
    [types_js_1.Network.CHILIZ_MAINNET]: 'chiliz-rpc',
    [types_js_1.Network.CHILIZ_SPICY]: 'chiliz-spicy-rpc',
    [types_js_1.Network.MOONBEAM_MAINNET]: 'moonbeam-rpc',
    [types_js_1.Network.BAHAMUT_MAINNET]: 'bahamut-rpc',
    [types_js_1.Network.TRON_MAINNET]: 'tron-evm-rpc',
    [types_js_1.Network.MOONRIVER_MAINNET]: 'moonriver-rpc',
};
//# sourceMappingURL=web3_provider_publicnode.js.map