import { NomicLabsHardhatPluginError } from "hardhat/plugins";
export declare class HardhatEthersError extends NomicLabsHardhatPluginError {
    constructor(message: string, parent?: Error);
}
export declare class NotImplementedError extends HardhatEthersError {
    constructor(method: string);
}
export declare class UnsupportedEventError extends HardhatEthersError {
    constructor(event: any);
}
export declare class AccountIndexOutOfRange extends HardhatEthersError {
    constructor(accountIndex: number, accountsLength: number);
}
export declare class BroadcastedTxDifferentHash extends HardhatEthersError {
    constructor(txHash: string, broadcastedTxHash: string);
}
//# sourceMappingURL=errors.d.ts.map