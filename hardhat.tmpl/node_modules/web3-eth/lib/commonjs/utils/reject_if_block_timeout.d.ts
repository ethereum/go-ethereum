import { EthExecutionAPI, Bytes } from 'web3-types';
import { Web3Context } from 'web3-core';
export interface ResourceCleaner {
    clean: () => void;
}
export declare function rejectIfBlockTimeout(web3Context: Web3Context<EthExecutionAPI>, transactionHash?: Bytes): Promise<[Promise<never>, ResourceCleaner]>;
