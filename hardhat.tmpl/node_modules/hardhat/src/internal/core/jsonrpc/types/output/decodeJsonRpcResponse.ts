import * as t from "io-ts";
import { PathReporter } from "io-ts/lib/PathReporter";

import { InvalidResponseError } from "../../../providers/errors";

/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */

/**
 * This function decodes an RPC out type, throwing InvalidResponseError if it's not valid.
 */
export function decodeJsonRpcResponse<T>(value: unknown, codec: t.Type<T>): T {
  const result = codec.decode(value);

  if (result.isLeft()) {
    throw new InvalidResponseError(
      `Invalid JSON-RPC response's result.

Errors: ${PathReporter.report(result).join(", ")}`
    );
  }

  return result.value;
}
