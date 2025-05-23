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
//
/**
 * An object holding the interface Ids of the ENS resolver contracts. Please see [how to write a resolver](https://docs.ens.domains/contract-developer-guide/writing-a-resolver).
 */
export const interfaceIds = {
    addr: '0x3b3b57de',
    name: '0x691f3431',
    abi: '0x2203ab56',
    pubkey: '0xc8690233',
    text: '0x59d1d43c',
    contenthash: '0xbc1c58d1',
};
/**
 * An object holding the functions that are supported by the ENS resolver contracts/interfaces.
 */
export const methodsInInterface = {
    setAddr: 'addr',
    addr: 'addr',
    setPubkey: 'pubkey',
    pubkey: 'pubkey',
    setContenthash: 'contenthash',
    contenthash: 'contenthash',
    text: 'text',
    name: 'name',
};
/**
 * An object holding the addressed of the ENS registries on the different networks (mainnet, goerli).
 */
export const registryAddresses = {
    main: '0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e',
    goerli: '0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e',
};
export const networkIds = {
    '0x1': 'main',
    '0x5': 'goerli',
};
//# sourceMappingURL=config.js.map