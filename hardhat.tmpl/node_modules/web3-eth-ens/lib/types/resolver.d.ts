import { Contract } from 'web3-eth-contract';
import { Address, PayableCallOptions } from 'web3-types';
import { PublicResolverAbi } from './abi/ens/PublicResolver.js';
import { Registry } from './registry.js';
export declare class Resolver {
    private readonly registry;
    constructor(registry: Registry);
    private getResolverContractAdapter;
    checkInterfaceSupport(resolverContract: Contract<typeof PublicResolverAbi>, methodName: string): Promise<void>;
    supportsInterface(ENSName: string, interfaceId: string): Promise<import("web3-types").MatchPrimitiveType<"bool", unknown>>;
    getAddress(ENSName: string, coinType?: number): Promise<import("web3-types").MatchPrimitiveType<"bytes", unknown>>;
    getPubkey(ENSName: string): Promise<unknown[] & Record<1, import("web3-types").MatchPrimitiveType<"bytes32", unknown>> & Record<0, import("web3-types").MatchPrimitiveType<"bytes32", unknown>> & [] & Record<"x", import("web3-types").MatchPrimitiveType<"bytes32", unknown>> & Record<"y", import("web3-types").MatchPrimitiveType<"bytes32", unknown>>>;
    getContenthash(ENSName: string): Promise<import("web3-types").MatchPrimitiveType<"bytes", unknown>>;
    setAddress(ENSName: string, address: Address, txConfig: PayableCallOptions): Promise<{
        readonly transactionHash: string;
        readonly transactionIndex: bigint;
        readonly blockHash: string;
        readonly blockNumber: bigint;
        readonly from: string;
        readonly to: string;
        readonly cumulativeGasUsed: bigint;
        readonly gasUsed: bigint;
        readonly effectiveGasPrice?: bigint | undefined;
        readonly contractAddress?: string | undefined;
        readonly logs: {
            readonly id?: string | undefined;
            readonly removed?: boolean | undefined;
            readonly logIndex?: bigint | undefined;
            readonly transactionIndex?: bigint | undefined;
            readonly transactionHash?: string | undefined;
            readonly blockHash?: string | undefined;
            readonly blockNumber?: bigint | undefined;
            readonly address?: string | undefined;
            readonly data?: string | undefined;
            readonly topics?: string[] | undefined;
        }[];
        readonly logsBloom: string;
        readonly root: string;
        readonly status: bigint;
        readonly type?: bigint | undefined;
        events?: {
            [x: string]: {
                readonly event: string;
                readonly id?: string | undefined;
                readonly logIndex?: bigint | undefined;
                readonly transactionIndex?: bigint | undefined;
                readonly transactionHash?: string | undefined;
                readonly blockHash?: string | undefined;
                readonly blockNumber?: bigint | undefined;
                readonly address: string;
                readonly topics: string[];
                readonly data: string;
                readonly raw?: {
                    data: string;
                    topics: unknown[];
                } | undefined;
                readonly returnValues: {
                    [x: string]: unknown;
                };
                readonly signature?: string | undefined;
            };
        } | undefined;
    }>;
    getText(ENSName: string, key: string): Promise<string>;
    getName(address: string, checkInterfaceSupport?: boolean): Promise<string>;
}
//# sourceMappingURL=resolver.d.ts.map