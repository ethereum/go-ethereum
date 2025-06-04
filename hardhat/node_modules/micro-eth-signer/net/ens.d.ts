import { type IWeb3Provider } from '../utils.ts';
export declare function namehash(address: string): Uint8Array;
export default class ENS {
    static ADDRESS_ZERO: string;
    static REGISTRY: string;
    static REGISTRY_CONTRACT: readonly [{
        readonly name: "resolver";
        readonly type: "function";
        readonly inputs: readonly [{
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly outputs: readonly [{
            readonly type: "address";
        }];
    }];
    static RESOLVER_CONTRACT: readonly [{
        readonly name: "addr";
        readonly type: "function";
        readonly inputs: readonly [{
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly outputs: readonly [{
            readonly type: "address";
        }];
    }, {
        readonly name: "name";
        readonly type: "function";
        readonly inputs: readonly [{
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly outputs: readonly [{
            readonly type: "string";
        }];
    }];
    readonly net: IWeb3Provider;
    constructor(net: IWeb3Provider);
    getResolver(name: string): Promise<string | undefined>;
    nameToAddress(name: string): Promise<string | undefined>;
    addressToName(address: string): Promise<string | undefined>;
}
//# sourceMappingURL=ens.d.ts.map