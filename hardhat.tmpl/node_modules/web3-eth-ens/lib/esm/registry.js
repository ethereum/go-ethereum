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
import { Contract } from 'web3-eth-contract';
import { ENSRegistryAbi } from './abi/ens/ENSRegistry.js';
import { PublicResolverAbi } from './abi/ens/PublicResolver.js';
import { registryAddresses } from './config.js';
import { namehash } from './utils.js';
export class Registry {
    constructor(context, customRegistryAddress) {
        this.contract = new Contract(ENSRegistryAbi, customRegistryAddress !== null && customRegistryAddress !== void 0 ? customRegistryAddress : registryAddresses.main, context);
        this.context = context;
    }
    getOwner(name) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                const result = this.contract.methods.owner(namehash(name)).call();
                return result;
            }
            catch (error) {
                throw new Error(); // TODO: TransactionRevertInstructionError Needs to be added after web3-eth call method is implemented
            }
        });
    }
    getTTL(name) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                return this.contract.methods.ttl(namehash(name)).call();
            }
            catch (error) {
                throw new Error(); // TODO: TransactionRevertInstructionError Needs to be added after web3-eth call method is implemented
            }
        });
    }
    recordExists(name) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                const promise = this.contract.methods.recordExists(namehash(name)).call();
                return promise;
            }
            catch (error) {
                throw new Error(); // TODO: TransactionRevertInstructionError Needs to be added after web3-eth call method is implemented
            }
        });
    }
    getResolver(name) {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                return this.contract.methods
                    .resolver(namehash(name))
                    .call()
                    .then(address => {
                    // address type is unknown, not sure why
                    if (typeof address === 'string') {
                        const contract = new Contract(PublicResolverAbi, address, this.context);
                        // TODO: set contract provider needs to be added when ens current provider
                        return contract;
                    }
                    throw new Error();
                });
            }
            catch (error) {
                throw new Error(); // TODO: TransactionRevertInstructionError Needs to be added after web3-eth call method is implemented
            }
        });
    }
    get events() {
        return this.contract.events;
    }
}
//# sourceMappingURL=registry.js.map