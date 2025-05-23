import { EIP1193Provider, RequestArguments } from "../../../types";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";
import { rpcQuantityToNumber } from "../jsonrpc/types/base-types";

import { ProviderWrapper } from "./wrapper";

export abstract class ProviderWrapperWithChainId extends ProviderWrapper {
  private _chainId: number | undefined;

  protected async _getChainId(): Promise<number> {
    if (this._chainId === undefined) {
      try {
        this._chainId = await this._getChainIdFromEthChainId();
      } catch {
        // If eth_chainId fails we default to net_version
        this._chainId = await this._getChainIdFromEthNetVersion();
      }
    }

    return this._chainId;
  }

  private async _getChainIdFromEthChainId(): Promise<number> {
    const id = (await this._wrappedProvider.request({
      method: "eth_chainId",
    })) as string;

    return rpcQuantityToNumber(id);
  }

  private async _getChainIdFromEthNetVersion(): Promise<number> {
    const id = (await this._wrappedProvider.request({
      method: "net_version",
    })) as string;

    // There's a node returning this as decimal instead of QUANTITY.
    // TODO: Document here which node does that
    return id.startsWith("0x") ? rpcQuantityToNumber(id) : parseInt(id, 10);
  }
}

export class ChainIdValidatorProvider extends ProviderWrapperWithChainId {
  private _alreadyValidated = false;

  constructor(
    provider: EIP1193Provider,
    private readonly _expectedChainId: number
  ) {
    super(provider);
  }

  public async request(args: RequestArguments): Promise<unknown> {
    if (!this._alreadyValidated) {
      const actualChainId = await this._getChainId();
      if (actualChainId !== this._expectedChainId) {
        throw new HardhatError(ERRORS.NETWORK.INVALID_GLOBAL_CHAIN_ID, {
          configChainId: this._expectedChainId,
          connectionChainId: actualChainId,
        });
      }

      this._alreadyValidated = true;
    }

    return this._wrappedProvider.request(args);
  }
}
