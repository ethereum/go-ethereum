import type { Provider as EdrProviderT } from "@nomicfoundation/edr";
import type { Address } from "@nomicfoundation/ethereumjs-util";
import type {
  MinimalEVMResult,
  MinimalInterpreterStep,
  MinimalMessage,
} from "./types";

import { AsyncEventEmitter } from "@nomicfoundation/ethereumjs-util";

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
    putContractStorage: (
      address: Address,
      slotHash: Buffer,
      slotValue: Buffer
    ) => Promise<void>;
  };
}

// we need to use a type instead of an interface to satisfy the type constraint
// of the AsyncEventEmitter type param
// eslint-disable-next-line @typescript-eslint/consistent-type-definitions
type MinimalEthereumJsVmEvents = {
  beforeTx: () => void;
  afterTx: () => void;
};

// we need to use a type instead of an interface to satisfy the type constraint
// of the AsyncEventEmitter type param
// eslint-disable-next-line @typescript-eslint/consistent-type-definitions
type MinimalEthereumJsEvmEvents = {
  beforeMessage: (
    data: MinimalMessage,
    resolve?: (result?: any) => void
  ) => void;
  afterMessage: (
    data: MinimalEVMResult,
    resolve?: (result?: any) => void
  ) => void;
  step: (
    data: MinimalInterpreterStep,
    resolve?: (result?: any) => void
  ) => void;
};

export class MinimalEthereumJsVmEventEmitter extends AsyncEventEmitter<MinimalEthereumJsVmEvents> {}
export class MinimalEthereumJsEvmEventEmitter extends AsyncEventEmitter<MinimalEthereumJsEvmEvents> {}

export function getMinimalEthereumJsVm(
  provider: EdrProviderT
): MinimalEthereumJsVm {
  const minimalEthereumJsVm: MinimalEthereumJsVm = {
    events: new MinimalEthereumJsVmEventEmitter(),
    evm: {
      events: new MinimalEthereumJsEvmEventEmitter(),
    },
    stateManager: {
      putContractCode: async (address: Address, code: Buffer) => {
        await provider.handleRequest(
          JSON.stringify({
            method: "hardhat_setCode",
            params: [address.toString(), `0x${code.toString("hex")}`],
          })
        );
      },
      getContractStorage: async (address: Address, slotHash: Buffer) => {
        const responseObject = await provider.handleRequest(
          JSON.stringify({
            method: "eth_getStorageAt",
            params: [address.toString(), `0x${slotHash.toString("hex")}`],
          })
        );

        const response = JSON.parse(responseObject.json);

        return Buffer.from(response.result.slice(2), "hex");
      },
      putContractStorage: async (
        address: Address,
        slotHash: Buffer,
        slotValue: Buffer
      ) => {
        await provider.handleRequest(
          JSON.stringify({
            method: "hardhat_setStorageAt",
            params: [
              address.toString(),
              `0x${slotHash.toString("hex")}`,
              `0x${slotValue.toString("hex")}`,
            ],
          })
        );
      },
    },
  };

  return minimalEthereumJsVm;
}
