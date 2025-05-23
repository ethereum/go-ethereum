import { Account, Address, privateToAddress, toBytes } from "@ethereumjs/util";

import { GenesisAccount } from "../node-types";

import { isHexPrefixed } from "./isHexPrefixed";

export function makeAccount(ga: GenesisAccount) {
  let balance: bigint;

  if (typeof ga.balance === "string" && isHexPrefixed(ga.balance)) {
    balance = BigInt(ga.balance);
  } else {
    balance = BigInt(ga.balance);
  }

  const account = Account.fromAccountData({ balance });
  const pk = toBytes(ga.privateKey);
  const address = new Address(privateToAddress(pk));
  return { account, address };
}
