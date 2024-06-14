import * as secp256k1 from "secp256k1";

export function privateKeyVerify(privateKey: Buffer): boolean {
  return secp256k1.privateKeyVerify(privateKey);
}

export function publicKeyCreate(privateKey: Buffer, compressed = true): Buffer {
  return Buffer.from(secp256k1.publicKeyCreate(privateKey, compressed));
}

export function publicKeyVerify(publicKey: Buffer): boolean {
  return secp256k1.publicKeyVerify(publicKey);
}

export function publicKeyConvert(publicKey: Buffer, compressed = true): Buffer {
  return Buffer.from(secp256k1.publicKeyConvert(publicKey, compressed));
}

export function privateKeyTweakAdd(publicKey: Buffer, tweak: Buffer): Buffer {
  return Buffer.from(
    secp256k1.privateKeyTweakAdd(Buffer.from(publicKey), tweak)
  );
}

export function publicKeyTweakAdd(
  publicKey: Buffer,
  tweak: Buffer,
  compressed = true
): Buffer {
  return Buffer.from(
    secp256k1.publicKeyTweakAdd(Buffer.from(publicKey), tweak, compressed)
  );
}

export function sign(
  message: Buffer,
  privateKey: Buffer
): { signature: Buffer; recovery: number } {
  const ret = secp256k1.ecdsaSign(message, privateKey);
  return { signature: Buffer.from(ret.signature), recovery: ret.recid };
}

export function verify(
  message: Buffer,
  signature: Buffer,
  publicKey: Buffer
): boolean {
  return secp256k1.ecdsaVerify(signature, message, publicKey);
}
