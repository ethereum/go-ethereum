const randombytes = require("randombytes");

export function getRandomBytes(bytes: number): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    randombytes(bytes, function(err: any, resp: Buffer) {
      if (err) {
        reject(err);
        return;
      }

      resolve(resp);
    });
  });
}

export function getRandomBytesSync(bytes: number): Buffer {
  return randombytes(bytes);
}
