const blake2bJs = require("blakejs");

export function blake2b(input: Buffer, outputLength = 64): Buffer {
  if (outputLength <= 0 || outputLength > 64) {
    throw Error("Invalid outputLength");
  }

  return Buffer.from(blake2bJs.blake2b(input, undefined, outputLength));
}
