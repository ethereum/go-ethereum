import { ExternallyOwnedAccount } from "@ethersproject/abstract-signer";
import { Bytes } from "@ethersproject/bytes";
import { Description } from "@ethersproject/properties";
export interface _CrowdsaleAccount {
    address: string;
    privateKey: string;
    _isCrowdsaleAccount: boolean;
}
export declare class CrowdsaleAccount extends Description<_CrowdsaleAccount> implements ExternallyOwnedAccount {
    readonly address: string;
    readonly privateKey: string;
    readonly mnemonic?: string;
    readonly path?: string;
    readonly _isCrowdsaleAccount: boolean;
    isCrowdsaleAccount(value: any): value is CrowdsaleAccount;
}
export declare function decrypt(json: string, password: Bytes | string): ExternallyOwnedAccount;
//# sourceMappingURL=crowdsale.d.ts.map