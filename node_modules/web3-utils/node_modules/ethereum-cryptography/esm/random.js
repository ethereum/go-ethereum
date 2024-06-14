import { randomBytes } from "@noble/hashes/utils";
export function getRandomBytesSync(bytes) {
    return randomBytes(bytes);
}
export async function getRandomBytes(bytes) {
    return randomBytes(bytes);
}
