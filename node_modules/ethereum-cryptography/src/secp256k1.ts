import { privateKeyVerify } from "secp256k1";
import { getRandomBytes, getRandomBytesSync } from "./random";

const SECP256K1_PRIVATE_KEY_SIZE = 32;

export async function createPrivateKey(): Promise<Uint8Array> {
  while (true) {
    const pk = await getRandomBytes(SECP256K1_PRIVATE_KEY_SIZE);
    if (privateKeyVerify(pk)) {
      return pk;
    }
  }
}

export function createPrivateKeySync(): Uint8Array {
  while (true) {
    const pk = getRandomBytesSync(SECP256K1_PRIVATE_KEY_SIZE);
    if (privateKeyVerify(pk)) {
      return pk;
    }
  }
}

export * from "secp256k1";
