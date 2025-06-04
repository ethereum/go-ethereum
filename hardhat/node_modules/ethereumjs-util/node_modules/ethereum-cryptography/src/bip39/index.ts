const bip39 = require("../vendor/bip39-without-wordlists");

export function generateMnemonic(
  wordlist: string[],
  strength: number = 128
): string {
  return bip39.generateMnemonic(strength, undefined, wordlist);
}

export function mnemonicToEntropy(
  mnemonic: string,
  wordlist: string[]
): Buffer {
  return bip39.mnemonicToEntropy(mnemonic, wordlist);
}

export function entropyToMnemonic(entropy: Buffer, wordlist: string[]): string {
  return bip39.entropyToMnemonic(entropy, wordlist);
}

export function validateMnemonic(
  mnemonic: string,
  wordlist: string[]
): boolean {
  return bip39.validateMnemonic(mnemonic, wordlist);
}

export async function mnemonicToSeed(
  mnemonic: string,
  passphrase: string = ""
): Promise<Buffer> {
  return bip39.mnemonicToSeed(mnemonic, passphrase);
}

export function mnemonicToSeedSync(
  mnemonic: string,
  passphrase: string = ""
): Buffer {
  return bip39.mnemonicToSeedSync(mnemonic, passphrase);
}
