import { Block, TransactionReceipt, TransactionResponse } from "@ethersproject/abstract-provider";
import { BigNumber } from "@ethersproject/bignumber";
import { AccessList } from "@ethersproject/transactions";
export declare type FormatFunc = (value: any) => any;
export declare type FormatFuncs = {
    [key: string]: FormatFunc;
};
export declare type Formats = {
    transaction: FormatFuncs;
    transactionRequest: FormatFuncs;
    receipt: FormatFuncs;
    receiptLog: FormatFuncs;
    block: FormatFuncs;
    blockWithTransactions: FormatFuncs;
    filter: FormatFuncs;
    filterLog: FormatFuncs;
};
export declare class Formatter {
    readonly formats: Formats;
    constructor();
    getDefaultFormats(): Formats;
    accessList(accessList: Array<any>): AccessList;
    number(number: any): number;
    type(number: any): number;
    bigNumber(value: any): BigNumber;
    boolean(value: any): boolean;
    hex(value: any, strict?: boolean): string;
    data(value: any, strict?: boolean): string;
    address(value: any): string;
    callAddress(value: any): string;
    contractAddress(value: any): string;
    blockTag(blockTag: any): string;
    hash(value: any, strict?: boolean): string;
    difficulty(value: any): number;
    uint256(value: any): string;
    _block(value: any, format: any): Block;
    block(value: any): Block;
    blockWithTransactions(value: any): Block;
    transactionRequest(value: any): any;
    transactionResponse(transaction: any): TransactionResponse;
    transaction(value: any): any;
    receiptLog(value: any): any;
    receipt(value: any): TransactionReceipt;
    topics(value: any): any;
    filter(value: any): any;
    filterLog(value: any): any;
    static check(format: {
        [name: string]: FormatFunc;
    }, object: any): any;
    static allowNull(format: FormatFunc, nullValue?: any): FormatFunc;
    static allowFalsish(format: FormatFunc, replaceValue: any): FormatFunc;
    static arrayOf(format: FormatFunc): FormatFunc;
}
export interface CommunityResourcable {
    isCommunityResource(): boolean;
}
export declare function isCommunityResourcable(value: any): value is CommunityResourcable;
export declare function isCommunityResource(value: any): boolean;
export declare function showThrottleMessage(): void;
//# sourceMappingURL=formatter.d.ts.map