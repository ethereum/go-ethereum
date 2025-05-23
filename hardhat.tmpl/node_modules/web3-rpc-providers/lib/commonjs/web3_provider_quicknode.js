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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.QuickNodeProvider = void 0;
const web3_errors_1 = require("web3-errors");
const types_js_1 = require("./types.js");
const web3_provider_js_1 = require("./web3_provider.js");
const errors_js_1 = require("./errors.js");
const isValid = (str) => str !== undefined && str.trim().length > 0;
class QuickNodeProvider extends web3_provider_js_1.Web3ExternalProvider {
    // eslint-disable-next-line default-param-last
    constructor(network = types_js_1.Network.ETH_MAINNET, transport = types_js_1.Transport.HTTPS, token = '', host = '', providerConfigOptions) {
        super(network, transport, token, host, providerConfigOptions);
    }
    request(payload, requestOptions) {
        const _super = Object.create(null, {
            request: { get: () => super.request }
        });
        return __awaiter(this, void 0, void 0, function* () {
            try {
                return yield _super.request.call(this, payload, requestOptions);
            }
            catch (error) {
                if (error instanceof web3_errors_1.ResponseError && error.statusCode === 429) {
                    throw new errors_js_1.QuickNodeRateLimitError(error);
                }
                throw error;
            }
        });
    }
    // eslint-disable-next-line class-methods-use-this
    getRPCURL(network, transport, _token, _host) {
        let host = '';
        let token = '';
        switch (network) {
            case types_js_1.Network.ETH_MAINNET:
                host = isValid(_host) ? _host : 'powerful-holy-bush.quiknode.pro';
                token = isValid(_token) ? _token : '3240624a343867035925ff7561eb60dfdba2a668';
                break;
            case types_js_1.Network.ETH_SEPOLIA:
                host = isValid(_host)
                    ? _host
                    : 'dimensional-fabled-glitter.ethereum-sepolia.quiknode.pro';
                token = isValid(_token) ? _token : '382a3b5a4b938f2d6e8686c19af4b22921fde2cd';
                break;
            case types_js_1.Network.ETH_HOLESKY:
                host = isValid(_host) ? _host : 'yolo-morning-card.ethereum-holesky.quiknode.pro';
                token = isValid(_token) ? _token : '481ebe70638c4dcf176af617a16d02ab866b9af9';
                break;
            case types_js_1.Network.ARBITRUM_MAINNET:
                host = isValid(_host)
                    ? _host
                    : 'autumn-divine-dinghy.arbitrum-mainnet.quiknode.pro';
                token = isValid(_token) ? _token : 'a5d7bfbf60b5ae9ce3628e53d69ef50d529e9a8c';
                break;
            case types_js_1.Network.ARBITRUM_SEPOLIA:
                host = isValid(_host) ? _host : 'few-patient-pond.arbitrum-sepolia.quiknode.pro';
                token = isValid(_token) ? _token : '3be985450970628c860b959c65cd2642dcafe53c';
                break;
            case types_js_1.Network.BNB_MAINNET:
                host = isValid(_host) ? _host : 'purple-empty-reel.bsc.quiknode.pro';
                token = isValid(_token) ? _token : 'ebf6c532961e21f092ff2facce1ec4c89c540158';
                break;
            case types_js_1.Network.BNB_TESTNET:
                host = isValid(_host) ? _host : 'floral-rough-scion.bsc-testnet.quiknode.pro';
                token = isValid(_token) ? _token : '5b297e5acff5f81f4c37ebf6f235f7299b6f9d28';
                break;
            case types_js_1.Network.POLYGON_MAINNET:
                host = isValid(_host) ? _host : 'small-chaotic-moon.matic.quiknode.pro';
                token = isValid(_token) ? _token : '847569f8a017e84d985e10d0f44365d965a951f1';
                break;
            case types_js_1.Network.POLYGON_AMOY:
                host = isValid(_host) ? _host : 'prettiest-side-shape.matic-amoy.quiknode.pro';
                token = isValid(_token) ? _token : '79a9476eea661d4f82de614db1d8a895b14b881c';
                break;
            default:
                throw new Error('Network info not avalible.');
        }
        return `${transport}://${host}/${token}`;
    }
}
exports.QuickNodeProvider = QuickNodeProvider;
//# sourceMappingURL=web3_provider_quicknode.js.map