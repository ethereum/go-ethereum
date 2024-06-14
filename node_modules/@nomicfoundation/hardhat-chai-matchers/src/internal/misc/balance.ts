import type { Addressable } from "ethers";

import { getAddressOf } from "./account";

export interface BalanceChangeOptions {
  includeFee?: boolean;
}

export function getAddresses(accounts: Array<Addressable | string>) {
  return Promise.all(accounts.map((account) => getAddressOf(account)));
}

export async function getBalances(
  accounts: Array<Addressable | string>,
  blockNumber?: number
): Promise<bigint[]> {
  const { toBigInt } = await import("ethers");
  const hre = await import("hardhat");
  const provider = hre.ethers.provider;

  return Promise.all(
    accounts.map(async (account) => {
      const address = await getAddressOf(account);
      const result = await provider.send("eth_getBalance", [
        address,
        `0x${blockNumber?.toString(16) ?? 0}`,
      ]);
      return toBigInt(result);
    })
  );
}
