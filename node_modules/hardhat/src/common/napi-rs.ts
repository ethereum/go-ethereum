import { HardhatError } from "../internal/core/errors";
import { ERRORS } from "../internal/core/errors-list";

export function requireNapiRsModule(id: string): unknown {
  try {
    return require(id);
  } catch (e: any) {
    if (e.code === "MODULE_NOT_FOUND") {
      throw new HardhatError(ERRORS.GENERAL.CORRUPTED_LOCKFILE);
    }

    // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
    throw e;
  }
}
