/// <reference types="node" />
/// <reference types="node" />
import type { Provider as EdrProviderT } from "@nomicfoundation/edr";
import type { Address } from "@ethereumjs/util";
import type { MinimalEVMResult, MinimalInterpreterStep, MinimalMessage } from "./types";
import { AsyncEventEmitter } from "@ethereumjs/util";
/**
 * Used by the provider to keep the `_vm` variable used by some plugins. This
 * interface only has the things used by those plugins.
 */
export interface MinimalEthereumJsVm {
    events: AsyncEventEmitter<MinimalEthereumJsVmEvents>;
    evm: {
        events: AsyncEventEmitter<MinimalEthereumJsEvmEvents>;
    };
    stateManager: {
        putContractCode: (address: Address, code: Buffer) => Promise<void>;
        getContractStorage: (address: Address, slotHash: Buffer) => Promise<Buffer>;
        putContractStorage: (address: Address, slotHash: Buffer, slotValue: Buffer) => Promise<void>;
    };
}
type MinimalEthereumJsVmEvents = {
    beforeTx: () => void;
    afterTx: () => void;
};
type MinimalEthereumJsEvmEvents = {
    beforeMessage: (data: MinimalMessage, resolve?: (result?: any) => void) => void;
    afterMessage: (data: MinimalEVMResult, resolve?: (result?: any) => void) => void;
    step: (data: MinimalInterpreterStep, resolve?: (result?: any) => void) => void;
};
export declare class MinimalEthereumJsVmEventEmitter extends AsyncEventEmitter<MinimalEthereumJsVmEvents> {
}
export declare class MinimalEthereumJsEvmEventEmitter extends AsyncEventEmitter<MinimalEthereumJsEvmEvents> {
}
export declare function getMinimalEthereumJsVm(provider: EdrProviderT): MinimalEthereumJsVm;
export {};
//# sourceMappingURL=minimal-vm.d.ts.map