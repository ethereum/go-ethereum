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
exports.Network = exports.Transport = void 0;
var Transport;
(function (Transport) {
    Transport["HTTPS"] = "https";
    Transport["WebSocket"] = "wss";
})(Transport || (exports.Transport = Transport = {}));
var Network;
(function (Network) {
    Network["ETH_MAINNET"] = "eth_mainnet";
    Network["ETH_SEPOLIA"] = "eth_sepolia";
    Network["ETH_HOLESKY"] = "eth_holesky";
    Network["POLYGON_MAINNET"] = "polygon_mainnet";
    Network["POLYGON_AMOY"] = "polygon_amoy";
    Network["AVALANCHE_C_MAINNET"] = "avalanche_c_mainnet";
    Network["AVALANCHE_P_MAINNET"] = "avalanche_p_mainnet";
    Network["AVALANCHE_X_MAINNET"] = "avalanche_x_mainnet";
    Network["ARBITRUM_MAINNET"] = "arbitrum_mainnet";
    Network["ARBITRUM_SEPOLIA"] = "arbitrum_sepolia";
    Network["BASE_MAINNET"] = "base_mainnet";
    Network["BASE_SEPOLIA"] = "base_sepolia";
    Network["OPTIMISM_MAINNET"] = "optimism_mainnet";
    Network["OPTIMISM_SEPOLIA"] = "optimism_sepolia";
    Network["FANTOM_MAINNET"] = "fantom_mainnet";
    Network["FANTOM_TESTNET"] = "fantom_testnet";
    Network["DYMENSION_MAINNET"] = "dymension_mainnet";
    Network["DYMENSION_TESTNET"] = "dymension_testnet";
    Network["BNB_MAINNET"] = "bnb_mainnet";
    Network["BNB_TESTNET"] = "bnb_testnet";
    Network["BSC_MAINNET"] = "bsc_mainnet";
    Network["BSC_TESTNET"] = "bsc_testnet";
    Network["ARBITRUM_ONE"] = "arbitrum_one";
    Network["ARBITRUM_NOVA"] = "arbitrum_nova";
    Network["AVALANCHE_FUJI_C"] = "avalanche_fuji_c";
    Network["AVALANCHE_FUJI_P"] = "avalanche_fuji_p";
    Network["AVALANCHE_FUJI_X"] = "avalanche_fuji_x";
    Network["BLAST_MAINNET"] = "blast_mainnet";
    Network["OPBNB_MAINNET"] = "opbnb_mainnet";
    Network["OPBNB_TESTNET"] = "opbnb_testnet";
    Network["GNOSIS_MAINNET"] = "gnosis_mainnet";
    Network["GNOSIS_CHIADO"] = "gnosis_chiado";
    Network["PULSECHAIN_MAINNET"] = "pulsechain_mainnet";
    Network["PULSECHAIN_TESTNET"] = "pulsechain_testnet";
    Network["KAVA_MAINNET"] = "kava_mainnet";
    Network["CRONOS_MAINNET"] = "cronos_mainnet";
    Network["MANTLE_MAINNET"] = "mantle_mainnet";
    Network["CHILIZ_MAINNET"] = "chiliz_mainnet";
    Network["CHILIZ_SPICY"] = "chiliz_spicy";
    Network["MOONBEAM_MAINNET"] = "moonbeam_mainnet";
    Network["TAIKO_MAINNET"] = "taiko_mainnet";
    Network["TAIKO_HEKLA"] = "taiko_hekla";
    Network["LINEA_MAINNET"] = "linea_mainnet";
    Network["LINEA_SEPOLIA"] = "linea_sepolia";
    Network["BAHAMUT_MAINNET"] = "bahamut_mainnet";
    Network["SCROLL_MAINNET"] = "scroll_mainnet";
    Network["SCROLL_SEPOLIA"] = "scroll_sepolia";
    Network["TRON_MAINNET"] = "tron_mainnet";
    Network["SYSCOIN_MAINNET"] = "syscoin_mainnet";
    Network["SYSCOIN_TANENBAUM"] = "syscoin_tanenbaum";
    Network["MOONRIVER_MAINNET"] = "moonriver_mainnet";
    Network["HAQQ_MAINNET"] = "haqq_mainnet";
    Network["EVMOS_MAINNET"] = "evmos_mainnet";
    Network["EVMOS_TESTNET"] = "evmos_testnet";
    Network["BERACHAIN_TESTNET"] = "berachain_testnet";
})(Network || (exports.Network = Network = {}));
//# sourceMappingURL=types.js.map