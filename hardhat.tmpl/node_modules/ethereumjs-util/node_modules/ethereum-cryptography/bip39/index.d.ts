/// <reference types="node" />
export declare function generateMnemonic(wordlist: string[], strength?: number): string;
export declare function mnemonicToEntropy(mnemonic: string, wordlist: string[]): Buffer;
export declare function entropyToMnemonic(entropy: Buffer, wordlist: string[]): string;
export declare function validateMnemonic(mnemonic: string, wordlist: string[]): boolean;
export declare function mnemonicToSeed(mnemonic: string, passphrase?: string): Promise<Buffer>;
export declare function mnemonicToSeedSync(mnemonic: string, passphrase?: string): Buffer;
//# sourceMappingURL=index.d.ts.map