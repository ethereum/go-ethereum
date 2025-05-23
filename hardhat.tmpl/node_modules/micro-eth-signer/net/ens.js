"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.namehash = namehash;
const sha3_1 = require("@noble/hashes/sha3");
const utils_1 = require("@noble/hashes/utils");
const decoder_ts_1 = require("../abi/decoder.js");
const utils_ts_1 = require("../utils.js");
// No support for IDN names
function namehash(address) {
    let res = new Uint8Array(32);
    if (!address)
        return res;
    for (let label of address.split('.').reverse())
        res = (0, sha3_1.keccak_256)((0, utils_1.concatBytes)(res, (0, sha3_1.keccak_256)(label)));
    return res;
}
class ENS {
    constructor(net) {
        this.net = net;
    }
    async getResolver(name) {
        const contract = (0, decoder_ts_1.createContract)(ENS.REGISTRY_CONTRACT, this.net, ENS.REGISTRY);
        const res = await contract.resolver.call(namehash(name));
        if (res === ENS.ADDRESS_ZERO)
            return;
        return res;
    }
    async nameToAddress(name) {
        const resolver = await this.getResolver(name);
        if (!resolver)
            return;
        const contract = (0, decoder_ts_1.createContract)(ENS.RESOLVER_CONTRACT, this.net, resolver);
        const addr = await contract.addr.call(namehash(name));
        if (addr === ENS.ADDRESS_ZERO)
            return;
        return addr;
    }
    async addressToName(address) {
        const addrDomain = `${(0, utils_ts_1.strip0x)(address).toLowerCase()}.addr.reverse`;
        const resolver = await this.getResolver(addrDomain);
        if (!resolver)
            return;
        const contract = (0, decoder_ts_1.createContract)(ENS.RESOLVER_CONTRACT, this.net, resolver);
        const name = await contract.name.call(namehash(addrDomain));
        if (!name)
            return;
        // From spec: ENS does not enforce accuracy of reverse records -
        // anyone may claim that the name for their address is 'alice.eth'.
        // To be certain the claim is accurate, you must always perform a forward
        // resolution for the returned name and check whether it matches the original address.
        const realAddr = await this.nameToAddress(name);
        if (realAddr !== address)
            return;
        return name;
    }
}
ENS.ADDRESS_ZERO = '0x0000000000000000000000000000000000000000';
ENS.REGISTRY = '0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e';
ENS.REGISTRY_CONTRACT = [
    {
        name: 'resolver',
        type: 'function',
        inputs: [{ name: 'node', type: 'bytes32' }],
        outputs: [{ type: 'address' }],
    },
];
ENS.RESOLVER_CONTRACT = [
    {
        name: 'addr',
        type: 'function',
        inputs: [{ name: 'node', type: 'bytes32' }],
        outputs: [{ type: 'address' }],
    },
    {
        name: 'name',
        type: 'function',
        inputs: [{ name: 'node', type: 'bytes32' }],
        outputs: [{ type: 'string' }],
    },
];
exports.default = ENS;
//# sourceMappingURL=ens.js.map