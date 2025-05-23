import { Web3Context, Web3ContextObject } from 'web3-core';
import { RevertInstructionError } from 'web3-errors';
import { Contract } from 'web3-eth-contract';
import { Address, EthExecutionAPI, PayableCallOptions, SupportedProviders, TransactionReceipt, Web3NetAPI } from 'web3-types';
import { PublicResolverAbi } from './abi/ens/PublicResolver.js';
/**
 * This class is designed to interact with the ENS system on the Ethereum blockchain.
 * For using ENS package, first install Web3 package using: `npm i web3` or `yarn add web3` based on your package manager, after that ENS features can be used as mentioned in following snippet.
 * ```ts
 *
 * import { Web3 } from 'web3';
 *
 * const web3 = new Web3('https://127.0.0.1:4545');
 *
 * console.log(await web3.eth.ens.getAddress('ethereum.eth'))
 * ```
 * For using individual package install `web3-eth-ens` packages using: `npm i web3-eth-ens` or `yarn add web3-eth-ens`. This is more efficient approach for building lightweight applications.
 *
 * ```ts
 *import { ENS } from 'web3-eth-ens';
 *
 * const ens = new ENS(undefined,'https://127.0.0.1:4545');
 *
 * console.log(await ens.getAddress('vitalik.eth'));
 * ```
 */
export declare class ENS extends Web3Context<EthExecutionAPI & Web3NetAPI> {
    /**
     * The registryAddress property can be used to define a custom registry address when you are connected to an unknown chain. It defaults to the main registry address.
     */
    registryAddress: string;
    private readonly _registry;
    private readonly _resolver;
    private _detectedAddress?;
    private _lastSyncCheck?;
    /**
     * Use to create an instance of ENS
     * @param registryAddr - (Optional) The address of the ENS registry (default: mainnet registry address)
     * @param provider - (Optional) The provider to use for the ENS instance
     * @example
     * ```ts
     * const ens = new ENS(
     * 	"0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e",
     * 	"http://localhost:8545"
     * );
     *
     * console.log( ens.defaultChain);
     * > mainnet
     * ```
     */
    constructor(registryAddr?: string, provider?: SupportedProviders<EthExecutionAPI & Web3NetAPI> | Web3ContextObject<EthExecutionAPI & Web3NetAPI> | string);
    /**
     * Returns the Resolver by the given address
     * @param name - The name of the ENS domain
     * @returns - An contract instance of the resolver
     *
     * @example
     * ```ts
     * const resolver = await ens.getResolver('resolver');
     *
     * console.log(resolver.options.address);
     * > '0x1234567890123456789012345678901234567890'
     * ```
     */
    getResolver(name: string): Promise<Contract<typeof PublicResolverAbi>>;
    /**
     * Returns true if the record exists
     * @param name - The ENS name
     * @returns - Returns `true` if node exists in this ENS registry. This will return `false` for records that are in the legacy ENS registry but have not yet been migrated to the new one.
     * @example
     * ```ts
     * const exists = await web3.eth.ens.recordExists('ethereum.eth');
     * ```
     */
    recordExists(name: string): Promise<unknown>;
    /**
     * Returns the caching TTL (time-to-live) of an ENS name.
     * @param name - The ENS name
     * @returns - Returns the caching TTL (time-to-live) of a name.
     * @example
     * ```ts
     * const owner = await web3.eth.ens.getTTL('ethereum.eth');
     * ```
     */
    getTTL(name: string): Promise<unknown>;
    /**
     * Returns the owner by the given name and current configured or detected Registry
     * @param name - The ENS name
     * @returns - Returns the address of the owner of the name.
     * @example
     * ```ts
     * const owner = await web3.eth.ens.getOwner('ethereum.eth');
     * ```
     */
    getOwner(name: string): Promise<unknown>;
    /**
     * Resolves an ENS name to an Ethereum address.
     * @param ENSName - The ENS name to resolve
     * @param coinType - (Optional) The coin type, defaults to 60 (ETH)
     * @returns - The Ethereum address of the given name
     * ```ts
     * const address = await web3.eth.ens.getAddress('ethereum.eth');
     * console.log(address);
     * > '0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359'
     * ```
     */
    getAddress(ENSName: string, coinType?: number): Promise<import("web3-types").MatchPrimitiveType<"bytes", unknown>>;
    /**
     * ERC-634 - Returns the text content stored in the resolver for the specified key.
     * @param ENSName - The ENS name to resolve
     * @param key - The key to resolve https://github.com/ethereum/ercs/blob/master/ERCS/erc-634.md#global-keys
     * @returns - The value content stored in the resolver for the specified key
     */
    getText(ENSNameOrAddr: string | Address, key: string): Promise<string>;
    /**
     * Resolves the name of an ENS node.
     * @param ENSName - The node to resolve
     * @returns - The name
     */
    getName(ENSName: string, checkInterfaceSupport?: boolean): Promise<string>;
    /**
     * Returns the X and Y coordinates of the curve point for the public key.
     * @param ENSName - The ENS name
     * @returns - The X and Y coordinates of the curve point for the public key
     * @example
     * ```ts
     * const key = await web3.eth.ens.getPubkey('ethereum.eth');
     * console.log(key);
     * > {
     * "0": "0x0000000000000000000000000000000000000000000000000000000000000000",
     * "1": "0x0000000000000000000000000000000000000000000000000000000000000000",
     * "x": "0x0000000000000000000000000000000000000000000000000000000000000000",
     * "y": "0x0000000000000000000000000000000000000000000000000000000000000000"
     * }
     * ```
     */
    getPubkey(ENSName: string): Promise<unknown[] & Record<1, import("web3-types").MatchPrimitiveType<"bytes32", unknown>> & Record<0, import("web3-types").MatchPrimitiveType<"bytes32", unknown>> & [] & Record<"x", import("web3-types").MatchPrimitiveType<"bytes32", unknown>> & Record<"y", import("web3-types").MatchPrimitiveType<"bytes32", unknown>>>;
    /**
     * Returns the content hash object associated with an ENS node.
     * @param ENSName - The ENS name
     * @returns - The content hash object associated with an ENS node
     * @example
     * ```ts
     * const hash = await web3.eth.ens.getContenthash('ethereum.eth');
     * console.log(hash);
     * > 'QmaEBknbGT4bTQiQoe2VNgBJbRfygQGktnaW5TbuKixjYL'
     * ```
     */
    getContenthash(ENSName: string): Promise<import("web3-types").MatchPrimitiveType<"bytes", unknown>>;
    /**
     * Checks if the current used network is synced and looks for ENS support there.
     * Throws an error if not.
     * @returns - The address of the ENS registry if the network has been detected successfully
     * @example
     * ```ts
     * console.log(await web3.eth.ens.checkNetwork());
     * > '0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e'
     * ```
     */
    checkNetwork(): Promise<string>;
    /**
     * Returns true if the related Resolver does support the given signature or interfaceId.
     * @param ENSName - The ENS name
     * @param interfaceId - The signature of the function or the interfaceId as described in the ENS documentation
     * @returns - `true` if the related Resolver does support the given signature or interfaceId.
     * @example
     * ```ts
     * const supports = await web3.eth.ens.supportsInterface('ethereum.eth', 'addr(bytes32');
     * console.log(supports);
     * > true
     * ```
     */
    supportsInterface(ENSName: string, interfaceId: string): Promise<import("web3-types").MatchPrimitiveType<"bool", unknown>>;
    /**
     * @returns - Returns all events that can be emitted by the ENS registry.
     */
    get events(): import("web3-eth-contract").ContractEventsInterface<readonly [{
        readonly anonymous: false;
        readonly inputs: readonly [{
            readonly indexed: true;
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }, {
            readonly indexed: true;
            readonly internalType: "bytes32";
            readonly name: "label";
            readonly type: "bytes32";
        }, {
            readonly indexed: false;
            readonly internalType: "address";
            readonly name: "owner";
            readonly type: "address";
        }];
        readonly name: "NewOwner";
        readonly type: "event";
    }, {
        readonly anonymous: false;
        readonly inputs: readonly [{
            readonly indexed: true;
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }, {
            readonly indexed: false;
            readonly internalType: "address";
            /**
             * This class is designed to interact with the ENS system on the Ethereum blockchain.
             * For using ENS package, first install Web3 package using: `npm i web3` or `yarn add web3` based on your package manager, after that ENS features can be used as mentioned in following snippet.
             * ```ts
             *
             * import { Web3 } from 'web3';
             *
             * const web3 = new Web3('https://127.0.0.1:4545');
             *
             * console.log(await web3.eth.ens.getAddress('ethereum.eth'))
             * ```
             * For using individual package install `web3-eth-ens` packages using: `npm i web3-eth-ens` or `yarn add web3-eth-ens`. This is more efficient approach for building lightweight applications.
             *
             * ```ts
             *import { ENS } from 'web3-eth-ens';
             *
             * const ens = new ENS(undefined,'https://127.0.0.1:4545');
             *
             * console.log(await ens.getAddress('vitalik.eth'));
             * ```
             */
            readonly name: "resolver";
            readonly type: "address";
        }];
        readonly name: "NewResolver";
        readonly type: "event";
    }, {
        readonly anonymous: false;
        readonly inputs: readonly [{
            readonly indexed: true;
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }, {
            readonly indexed: false;
            readonly internalType: "address";
            readonly name: "owner";
            readonly type: "address";
        }];
        readonly name: "Transfer";
        readonly type: "event";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "address";
            readonly name: "owner";
            readonly type: "address";
        }, {
            readonly internalType: "address";
            readonly name: "operator";
            readonly type: "address";
        }];
        readonly name: "isApprovedForAll";
        readonly outputs: readonly [{
            readonly internalType: "bool";
            readonly name: "";
            readonly type: "bool";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly name: "owner";
        readonly outputs: readonly [{
            readonly internalType: "address";
            readonly name: "";
            readonly type: "address";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly name: "recordExists";
        readonly outputs: readonly [{
            readonly internalType: "bool";
            readonly name: "";
            readonly type: "bool";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly name: "resolver";
        readonly outputs: readonly [{
            readonly internalType: "address";
            readonly name: "";
            readonly type: "address";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly name: "ttl";
        readonly outputs: readonly [{
            readonly internalType: "uint64";
            readonly name: "";
            readonly type: "uint64";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }], import("web3-types").ContractEvents<readonly [{
        readonly anonymous: false;
        readonly inputs: readonly [{
            readonly indexed: true;
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }, {
            readonly indexed: true;
            readonly internalType: "bytes32";
            readonly name: "label";
            readonly type: "bytes32";
        }, {
            readonly indexed: false;
            readonly internalType: "address";
            readonly name: "owner";
            readonly type: "address";
        }];
        readonly name: "NewOwner";
        readonly type: "event";
    }, {
        readonly anonymous: false;
        readonly inputs: readonly [{
            readonly indexed: true;
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }, {
            readonly indexed: false;
            readonly internalType: "address";
            /**
             * This class is designed to interact with the ENS system on the Ethereum blockchain.
             * For using ENS package, first install Web3 package using: `npm i web3` or `yarn add web3` based on your package manager, after that ENS features can be used as mentioned in following snippet.
             * ```ts
             *
             * import { Web3 } from 'web3';
             *
             * const web3 = new Web3('https://127.0.0.1:4545');
             *
             * console.log(await web3.eth.ens.getAddress('ethereum.eth'))
             * ```
             * For using individual package install `web3-eth-ens` packages using: `npm i web3-eth-ens` or `yarn add web3-eth-ens`. This is more efficient approach for building lightweight applications.
             *
             * ```ts
             *import { ENS } from 'web3-eth-ens';
             *
             * const ens = new ENS(undefined,'https://127.0.0.1:4545');
             *
             * console.log(await ens.getAddress('vitalik.eth'));
             * ```
             */
            readonly name: "resolver";
            readonly type: "address";
        }];
        readonly name: "NewResolver";
        readonly type: "event";
    }, {
        readonly anonymous: false;
        readonly inputs: readonly [{
            readonly indexed: true;
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }, {
            readonly indexed: false;
            readonly internalType: "address";
            readonly name: "owner";
            readonly type: "address";
        }];
        readonly name: "Transfer";
        readonly type: "event";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "address";
            readonly name: "owner";
            readonly type: "address";
        }, {
            readonly internalType: "address";
            readonly name: "operator";
            readonly type: "address";
        }];
        readonly name: "isApprovedForAll";
        readonly outputs: readonly [{
            readonly internalType: "bool";
            readonly name: "";
            readonly type: "bool";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly name: "owner";
        readonly outputs: readonly [{
            readonly internalType: "address";
            readonly name: "";
            readonly type: "address";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly name: "recordExists";
        readonly outputs: readonly [{
            readonly internalType: "bool";
            readonly name: "";
            readonly type: "bool";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly name: "resolver";
        readonly outputs: readonly [{
            readonly internalType: "address";
            readonly name: "";
            readonly type: "address";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }, {
        readonly inputs: readonly [{
            readonly internalType: "bytes32";
            readonly name: "node";
            readonly type: "bytes32";
        }];
        readonly name: "ttl";
        readonly outputs: readonly [{
            readonly internalType: "uint64";
            readonly name: "";
            readonly type: "uint64";
        }];
        readonly stateMutability: "view";
        readonly type: "function";
    }]>>;
    /**
     * Sets the address of an ENS name in his resolver.
     * @param name - The ENS name
     * @param address - The address to set
     * @param txConfig - (Optional) The transaction config
     * @returns - The transaction receipt
     * ```ts
     * const receipt = await ens.setAddress('web3js.eth','0xe2597eb05cf9a87eb1309e86750c903ec38e527e');
     *```
     */
    setAddress(name: string, address: Address, txConfig: PayableCallOptions): Promise<TransactionReceipt | RevertInstructionError>;
}
