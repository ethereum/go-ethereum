import type { mnemonicToSeedSync as mnemonicToSeedSyncT } from "ethereum-cryptography/bip39";
import type { HDKey as HDKeyT } from "ethereum-cryptography/hdkey";

export function deriveKeyFromMnemonicAndPath(
  mnemonic: string,
  hdPath: string,
  passphrase: string
): Buffer | undefined {
  const {
    mnemonicToSeedSync,
  }: {
    mnemonicToSeedSync: typeof mnemonicToSeedSyncT;
  } = require("ethereum-cryptography/bip39");
  // NOTE: If mnemonic has space or newline at the beginning or end, it will be trimmed.
  // This is because mnemonic containing them may generate different private keys.
  const trimmedMnemonic = mnemonic.trim();
  const seed = mnemonicToSeedSync(trimmedMnemonic, passphrase);

  const {
    HDKey,
  }: {
    HDKey: typeof HDKeyT;
  } = require("ethereum-cryptography/hdkey");

  const masterKey = HDKey.fromMasterSeed(seed);
  const derived = masterKey.derive(hdPath);

  return derived.privateKey === null
    ? undefined
    : Buffer.from(derived.privateKey);
}
