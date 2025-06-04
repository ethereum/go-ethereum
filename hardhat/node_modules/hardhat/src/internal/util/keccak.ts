import createKeccakHash from "keccak";

export function keccak256(data: Uint8Array): Uint8Array {
  return createKeccakHash("keccak256").update(Buffer.from(data)).digest();
}
