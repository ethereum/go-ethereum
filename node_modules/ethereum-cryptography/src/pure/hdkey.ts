export interface Versions {
  private: number;
  public: number;
}

export declare class HDKeyT {
  public static HARDENED_OFFSET: number;
  public static fromMasterSeed(seed: Buffer, versions?: Versions): HDKeyT;
  public static fromExtendedKey(base58key: string, versions?: Versions): HDKeyT;
  public static fromJSON(json: { xpriv: string }): HDKeyT;

  public versions: Versions;
  public depth: number;
  public index: number;
  public chainCode: Buffer | null;
  public privateKey: Buffer | null;
  public publicKey: Buffer | null;
  public fingerprint: number;
  public parentFingerprint: number;
  public pubKeyHash: Buffer | undefined;
  public identifier: Buffer | undefined;
  public privateExtendedKey: string;
  public publicExtendedKey: string;

  private constructor(versios: Versions);
  public derive(path: string): HDKeyT;
  public deriveChild(index: number): HDKeyT;
  public sign(hash: Buffer): Buffer;
  public verify(hash: Buffer, signature: Buffer): boolean;
  public wipePrivateData(): this;
  public toJSON(): { xpriv: string; xpub: string };
}

const hdkey: typeof HDKeyT = require("./vendor/hdkey-without-crypto");

export const HDKey = hdkey;
