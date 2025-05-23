import { Web3RequestManager } from 'web3-core';
import { Web3NetAPI } from 'web3-types';
export declare function getId(requestManager: Web3RequestManager<Web3NetAPI>): Promise<string>;
export declare function getPeerCount(requestManager: Web3RequestManager<Web3NetAPI>): Promise<string>;
export declare function isListening(requestManager: Web3RequestManager<Web3NetAPI>): Promise<boolean>;
