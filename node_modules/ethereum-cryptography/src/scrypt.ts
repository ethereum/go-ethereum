import "scrypt-js/thirdparty/setImmediate";
const scryptJs = require("scrypt-js");

export async function scrypt(
  password: Buffer,
  salt: Buffer,
  n: number,
  p: number,
  r: number,
  dklen: number
): Promise<Buffer> {
  return Buffer.from(await scryptJs.scrypt(password, salt, n, r, p, dklen));
}

export function scryptSync(
  password: Buffer,
  salt: Buffer,
  n: number,
  p: number,
  r: number,
  dklen: number
): Buffer {
  return Buffer.from(scryptJs.syncScrypt(password, salt, n, r, p, dklen));
}
