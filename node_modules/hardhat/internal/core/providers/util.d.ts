/// <reference types="node" />
import { HardhatNetworkAccountConfig, HardhatNetworkAccountsConfig } from "../../../types";
export declare function derivePrivateKeys(mnemonic: string, hdpath: string, initialIndex: number, count: number, passphrase: string): Buffer[];
export declare function normalizeHardhatNetworkAccountsConfig(accountsConfig: HardhatNetworkAccountsConfig): HardhatNetworkAccountConfig[];
//# sourceMappingURL=util.d.ts.map