import type { Addressable } from "ethers";

import assert from "assert";

import { HardhatChaiMatchersAssertionError } from "../errors";

export async function getAddressOf(
  account: Addressable | string
): Promise<string> {
  const { isAddressable } = await import("ethers");

  if (typeof account === "string") {
    assert(/^0x[0-9a-fA-F]{40}$/.test(account), `Invalid address ${account}`);
    return account;
  }

  if (isAddressable(account)) {
    return account.getAddress();
  }

  throw new HardhatChaiMatchersAssertionError(
    `Expected string or addressable, got ${account as any}`
  );
}
