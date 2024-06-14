import { ripemd160 } from "../ripemd160";
import { sha256 } from "../sha256";

export const createHmac = require("create-hmac");
export const randomBytes = require("randombytes");

class Hash {
  private buffers: Buffer[] = [];

  constructor(private readonly hashFunction: (msg: Buffer) => Buffer) {}

  public update(buffer: Buffer): this {
    if (!Buffer.isBuffer(buffer)) {
      throw new Error("hdkey-crypto shim is outdated");
    }

    this.buffers.push(buffer);

    return this;
  }

  public digest(param: any): Buffer {
    if (param) {
      throw new Error("hdkey-crypto shim is outdated");
    }

    return this.hashFunction(Buffer.concat(this.buffers));
  }
}

// We don't use create-hash here, as it doesn't work well with Rollup
export const createHash = (name: string) => {
  if (name === "ripemd160") {
    return new Hash(ripemd160);
  }

  if (name === "sha256") {
    return new Hash(sha256);
  }

  throw new Error("hdkey-crypto shim is outdated");
};
