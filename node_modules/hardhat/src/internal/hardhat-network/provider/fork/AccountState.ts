import { Map as ImmutableMap, Record as ImmutableRecord } from "immutable";

export interface AccountState {
  nonce: string | undefined;
  balance: string | undefined;
  // a null value means that the slot was set to 0 (i.e. deleted)
  storage: ImmutableMap<string, string | null>;
  code: string | undefined;
  storageCleared: boolean;
}

export const makeAccountState = ImmutableRecord<AccountState>({
  nonce: undefined,
  balance: undefined,
  storage: ImmutableMap<string, string | null>(),
  code: undefined,
  storageCleared: false,
});

// used for deleted accounts
// they need real values to avoid fetching the data from the remote node
export const makeEmptyAccountState = ImmutableRecord<AccountState>({
  nonce: "0x0",
  balance: "0x0",
  storage: ImmutableMap<string, string | null>(),
  code: "0x",
  storageCleared: true,
});
