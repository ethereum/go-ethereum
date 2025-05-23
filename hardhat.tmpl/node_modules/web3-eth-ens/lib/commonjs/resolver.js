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
exports.Resolver = void 0;
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const web3_validator_1 = require("web3-validator");
const config_js_1 = require("./config.js");
const utils_js_1 = require("./utils.js");
//  Default public resolver
//  https://github.com/ensdomains/resolvers/blob/master/contracts/PublicResolver.sol
class Resolver {
    constructor(registry) {
        this.registry = registry;
    }
    getResolverContractAdapter(ENSName) {
        return __awaiter(this, void 0, void 0, function* () {
            //  TODO : (Future 4.1.0 TDB) cache resolver contract if frequently queried same ENS name, refresh cache based on TTL and usage, also limit cache size, optional cache with a flag
            return this.registry.getResolver(ENSName);
        });
    }
    //  https://eips.ethereum.org/EIPS/eip-165
    // eslint-disable-next-line class-methods-use-this
    checkInterfaceSupport(resolverContract, methodName) {
        var _a, _b;
        return __awaiter(this, void 0, void 0, function* () {
            if ((0, web3_utils_1.isNullish)(config_js_1.interfaceIds[methodName]))
                throw new web3_errors_1.ResolverMethodMissingError((_a = resolverContract.options.address) !== null && _a !== void 0 ? _a : '', methodName);
            const supported = yield resolverContract.methods
                .supportsInterface(config_js_1.interfaceIds[methodName])
                .call();
            if (!supported)
                throw new web3_errors_1.ResolverMethodMissingError((_b = resolverContract.options.address) !== null && _b !== void 0 ? _b : '', methodName);
        });
    }
    supportsInterface(ENSName, interfaceId) {
        var _a;
        return __awaiter(this, void 0, void 0, function* () {
            const resolverContract = yield this.getResolverContractAdapter(ENSName);
            let interfaceIdParam = interfaceId;
            if (!(0, web3_validator_1.isHexStrict)(interfaceIdParam)) {
                interfaceIdParam = (_a = (0, web3_utils_1.sha3)(interfaceId)) !== null && _a !== void 0 ? _a : '';
                if (interfaceId === '')
                    throw new Error('Invalid interface Id');
                interfaceIdParam = interfaceIdParam.slice(0, 10);
            }
            return resolverContract.methods.supportsInterface(interfaceIdParam).call();
        });
    }
    // eslint-disable-next-line @typescript-eslint/no-inferrable-types
    getAddress(ENSName, coinType = 60) {
        return __awaiter(this, void 0, void 0, function* () {
            const resolverContract = yield this.getResolverContractAdapter(ENSName);
            yield this.checkInterfaceSupport(resolverContract, config_js_1.methodsInInterface.addr);
            return resolverContract.methods.addr((0, utils_js_1.namehash)(ENSName), coinType).call();
        });
    }
    getPubkey(ENSName) {
        return __awaiter(this, void 0, void 0, function* () {
            const resolverContract = yield this.getResolverContractAdapter(ENSName);
            yield this.checkInterfaceSupport(resolverContract, config_js_1.methodsInInterface.pubkey);
            return resolverContract.methods.pubkey((0, utils_js_1.namehash)(ENSName)).call();
        });
    }
    getContenthash(ENSName) {
        return __awaiter(this, void 0, void 0, function* () {
            const resolverContract = yield this.getResolverContractAdapter(ENSName);
            yield this.checkInterfaceSupport(resolverContract, config_js_1.methodsInInterface.contenthash);
            return resolverContract.methods.contenthash((0, utils_js_1.namehash)(ENSName)).call();
        });
    }
    setAddress(ENSName, address, txConfig) {
        return __awaiter(this, void 0, void 0, function* () {
            const resolverContract = yield this.getResolverContractAdapter(ENSName);
            yield this.checkInterfaceSupport(resolverContract, config_js_1.methodsInInterface.setAddr);
            return resolverContract.methods
                .setAddr((0, utils_js_1.namehash)(ENSName), address)
                .send(txConfig);
        });
    }
    getText(ENSName, key) {
        return __awaiter(this, void 0, void 0, function* () {
            const resolverContract = yield this.getResolverContractAdapter(ENSName);
            yield this.checkInterfaceSupport(resolverContract, config_js_1.methodsInInterface.text);
            return resolverContract.methods
                .text((0, utils_js_1.namehash)(ENSName), key).call();
        });
    }
    getName(address, checkInterfaceSupport = true) {
        return __awaiter(this, void 0, void 0, function* () {
            const reverseName = `${address.toLowerCase().substring(2)}.addr.reverse`;
            const resolverContract = yield this.getResolverContractAdapter(reverseName);
            if (checkInterfaceSupport)
                yield this.checkInterfaceSupport(resolverContract, config_js_1.methodsInInterface.name);
            return resolverContract.methods
                .name((0, utils_js_1.namehash)(reverseName)).call();
        });
    }
}
exports.Resolver = Resolver;
//# sourceMappingURL=resolver.js.map