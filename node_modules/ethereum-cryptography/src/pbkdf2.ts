import * as pbkdf2Js from "pbkdf2";

export function pbkdf2(
  password: Buffer,
  salt: Buffer,
  iterations: number,
  keylen: number,
  digest: string
): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    pbkdf2Js.pbkdf2(
      password,
      salt,
      iterations,
      keylen,
      digest,
      (err, result) => {
        if (err) {
          reject(err);
          return;
        }

        resolve(result);
      }
    );
  });
}

export function pbkdf2Sync(
  password: Buffer,
  salt: Buffer,
  iterations: number,
  keylen: number,
  digest: string
): Buffer {
  return pbkdf2Js.pbkdf2Sync(password, salt, iterations, keylen, digest);
}
