import { Web3Context } from 'web3-core';
import { DataFormat, EthExecutionAPI, Numbers, FilterParams } from 'web3-types';
/**
 * View additional documentations here: {@link Web3Eth.createNewPendingTransactionFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param returnFormat ({@link DataFormat}) Return format
 */
export declare function createNewPendingTransactionFilter<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.createNewFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filter ({@link FilterParam}) Filter param optional having from-block to-block address or params
 * @param returnFormat ({@link DataFormat}) Return format
 */
export declare function createNewFilter<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, filter: FilterParams, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.createNewBlockFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param returnFormat ({@link DataFormat}) Return format
 */
export declare function createNewBlockFilter<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
/**
 * View additional documentations here: {@link Web3Eth.uninstallFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filterIdentifier ({@link Numbers}) filter id
 */
export declare function uninstallFilter(web3Context: Web3Context<EthExecutionAPI>, filterIdentifier: Numbers): Promise<boolean>;
/**
 * View additional documentations here: {@link Web3Eth.getFilterChanges}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filterIdentifier ({@link Numbers}) filter id
 */
export declare function getFilterChanges<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, filterIdentifier: Numbers, returnFormat: ReturnFormat): Promise<(string | {
    readonly id?: string | undefined;
    readonly removed?: boolean | undefined;
    readonly logIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly transactionIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly transactionHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly blockHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly blockNumber?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly address?: import("web3-types").Address | undefined;
    readonly data?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly topics?: import("web3-types").ByteTypes[ReturnFormat["bytes"]][] | undefined;
})[]>;
/**
 * View additional documentations here: {@link Web3Eth.getFilterLogs}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filterIdentifier ({@link Numbers}) filter id
 */
export declare function getFilterLogs<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, filterIdentifier: Numbers, returnFormat: ReturnFormat): Promise<(string | {
    readonly id?: string | undefined;
    readonly removed?: boolean | undefined;
    readonly logIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly transactionIndex?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly transactionHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly blockHash?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly blockNumber?: import("web3-types").NumberTypes[ReturnFormat["number"]] | undefined;
    readonly address?: import("web3-types").Address | undefined;
    readonly data?: import("web3-types").ByteTypes[ReturnFormat["bytes"]] | undefined;
    readonly topics?: import("web3-types").ByteTypes[ReturnFormat["bytes"]][] | undefined;
})[]>;
//# sourceMappingURL=filtering_rpc_method_wrappers.d.ts.map