import { InternalError } from "../../../core/providers/errors";

export function assertHardhatNetworkInvariant(
  invariant: boolean,
  description: string
): asserts invariant {
  if (!invariant) {
    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw new InternalError(
      `Internal Hardhat Network invariant was violated: ${description}`
    );
  }
}
