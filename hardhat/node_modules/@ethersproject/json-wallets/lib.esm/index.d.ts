import { Bytes } from "@ethersproject/bytes";
import { ExternallyOwnedAccount } from "@ethersproject/abstract-signer";
import { decrypt as decryptCrowdsale } from "./crowdsale";
import { getJsonWalletAddress, isCrowdsaleWallet, isKeystoreWallet } from "./inspect";
import { decrypt as decryptKeystore, decryptSync as decryptKeystoreSync, encrypt as encryptKeystore, EncryptOptions, ProgressCallback } from "./keystore";
declare function decryptJsonWallet(json: string, password: Bytes | string, progressCallback?: ProgressCallback): Promise<ExternallyOwnedAccount>;
declare function decryptJsonWalletSync(json: string, password: Bytes | string): ExternallyOwnedAccount;
export { decryptCrowdsale, decryptKeystore, decryptKeystoreSync, encryptKeystore, isCrowdsaleWallet, isKeystoreWallet, getJsonWalletAddress, decryptJsonWallet, decryptJsonWalletSync, ProgressCallback, EncryptOptions, };
//# sourceMappingURL=index.d.ts.map