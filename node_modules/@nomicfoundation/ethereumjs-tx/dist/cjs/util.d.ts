import type { AccessList, AccessListBytes, TransactionType } from './types.js';
import type { Common } from '@nomicfoundation/ethereumjs-common';
export declare function checkMaxInitCodeSize(common: Common, length: number): void;
export declare class AccessLists {
    static getAccessListData(accessList: AccessListBytes | AccessList): {
        AccessListJSON: AccessList;
        accessList: AccessListBytes;
    };
    static verifyAccessList(accessList: AccessListBytes): void;
    static getAccessListJSON(accessList: AccessListBytes): any[];
    static getDataFeeEIP2930(accessList: AccessListBytes, common: Common): number;
}
export declare function txTypeBytes(txType: TransactionType): Uint8Array;
//# sourceMappingURL=util.d.ts.map