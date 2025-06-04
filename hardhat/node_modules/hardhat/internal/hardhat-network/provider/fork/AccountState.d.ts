import { Map as ImmutableMap, Record as ImmutableRecord } from "immutable";
export interface AccountState {
    nonce: string | undefined;
    balance: string | undefined;
    storage: ImmutableMap<string, string | null>;
    code: string | undefined;
    storageCleared: boolean;
}
export declare const makeAccountState: ImmutableRecord.Factory<AccountState>;
export declare const makeEmptyAccountState: ImmutableRecord.Factory<AccountState>;
//# sourceMappingURL=AccountState.d.ts.map