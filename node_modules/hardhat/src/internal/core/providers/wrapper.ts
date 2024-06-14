import { EIP1193Provider, RequestArguments } from "../../../types";
import { EventEmitterWrapper } from "../../util/event-emitter";

import { InvalidInputError } from "./errors";

/**
 * A wrapper class that makes it easy to implement the EIP1193 (Javascript Ethereum Provider) standard.
 * It comes baked in with all EventEmitter methods needed,
 * which will be added to the provider supplied in the constructor.
 * It also provides the interface for the standard .request() method as an abstract method.
 */
export abstract class ProviderWrapper
  extends EventEmitterWrapper
  implements EIP1193Provider
{
  constructor(protected readonly _wrappedProvider: EIP1193Provider) {
    super(_wrappedProvider);
  }

  public abstract request(args: RequestArguments): Promise<unknown>;

  /**
   * Extract the params from RequestArguments and optionally type them.
   * It defaults to an empty array if no params are found.
   */
  protected _getParams<ParamsT extends any[] = any[]>(
    args: RequestArguments
  ): ParamsT | [] {
    const params = args.params;

    if (params === undefined) {
      return [];
    }

    if (!Array.isArray(params)) {
      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw new InvalidInputError(
        "Hardhat Network doesn't support JSON-RPC params sent as an object"
      );
    }

    return params as ParamsT;
  }
}
