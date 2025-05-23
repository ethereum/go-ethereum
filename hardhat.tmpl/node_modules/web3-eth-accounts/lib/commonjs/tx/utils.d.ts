import { HexString } from 'web3-types';
import type { AccessList, AccessListUint8Array } from './types.js';
import type { Common } from '../common/common.js';
export declare const checkMaxInitCodeSize: (common: Common, length: number) => void;
export declare const getAccessListData: (accessList: AccessListUint8Array | AccessList) => {
    AccessListJSON: AccessList;
    accessList: AccessListUint8Array;
};
export declare const verifyAccessList: (accessList: AccessListUint8Array) => void;
export declare const getAccessListJSON: (accessList: AccessListUint8Array) => {
    address: HexString;
    storageKeys: HexString[];
}[];
export declare const getDataFeeEIP2930: (accessList: AccessListUint8Array, common: Common) => number;
