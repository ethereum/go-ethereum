import { Web3Context } from 'web3-core';
import { DataFormat, Web3NetAPI } from 'web3-types';
export declare function getId<ReturnFormat extends DataFormat>(web3Context: Web3Context<Web3NetAPI>, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
export declare function getPeerCount<ReturnFormat extends DataFormat>(web3Context: Web3Context<Web3NetAPI>, returnFormat: ReturnFormat): Promise<import("web3-types").NumberTypes[ReturnFormat["number"]]>;
export declare const isListening: (web3Context: Web3Context<Web3NetAPI>) => Promise<boolean>;
