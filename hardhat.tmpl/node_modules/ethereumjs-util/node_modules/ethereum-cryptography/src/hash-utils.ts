import { Hash } from "crypto";

export function createHashFunction(
  hashConstructor: () => Hash
): (msg: Buffer) => Buffer {
  return msg => {
    const hash = hashConstructor();
    hash.update(msg);
    return Buffer.from(hash.digest());
  };
}
