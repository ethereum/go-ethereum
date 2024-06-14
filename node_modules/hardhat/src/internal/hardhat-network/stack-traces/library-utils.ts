import { CompilerOutputBytecode } from "../../../types";

import { Opcode } from "./opcodes";

export function getLibraryAddressPositions(
  bytecodeOutput: CompilerOutputBytecode
): number[] {
  const positions = [];
  for (const libs of Object.values(bytecodeOutput.linkReferences)) {
    for (const references of Object.values(libs)) {
      for (const ref of references) {
        positions.push(ref.start);
      }
    }
  }

  return positions;
}

export function normalizeCompilerOutputBytecode(
  compilerOutputBytecodeObject: string,
  addressesPositions: number[]
): Buffer {
  const ZERO_ADDRESS = "0000000000000000000000000000000000000000";
  for (const position of addressesPositions) {
    compilerOutputBytecodeObject = linkHexStringBytecode(
      compilerOutputBytecodeObject,
      ZERO_ADDRESS,
      position
    );
  }

  return Buffer.from(compilerOutputBytecodeObject, "hex");
}

export function linkHexStringBytecode(
  code: string,
  address: string,
  position: number
) {
  if (address.startsWith("0x")) {
    address = address.substring(2);
  }

  return (
    code.substring(0, position * 2) +
    address +
    code.slice(position * 2 + address.length)
  );
}

export function zeroOutAddresses(
  code: Uint8Array,
  addressesPositions: number[]
): Uint8Array {
  const addressesSlices = addressesPositions.map((start) => ({
    start,
    length: 20,
  }));

  return zeroOutSlices(code, addressesSlices);
}

export function zeroOutSlices(
  code: Uint8Array,
  slices: Array<{ start: number; length: number }>
): Uint8Array {
  for (const { start, length } of slices) {
    code = Buffer.concat([
      code.slice(0, start),
      Buffer.alloc(length, 0),
      code.slice(start + length),
    ]);
  }

  return code;
}

export function normalizeLibraryRuntimeBytecodeIfNecessary(
  code: Uint8Array
): Uint8Array {
  // Libraries' protection normalization:
  // Solidity 0.4.20 introduced a protection to prevent libraries from being called directly.
  // This is done by modifying the code on deployment, and hard-coding the contract address.
  // The first instruction is a PUSH20 of the address, which we zero-out as a way of normalizing
  // it. Note that it's also zeroed-out in the compiler output.
  if (code[0] === Opcode.PUSH20) {
    return zeroOutAddresses(code, [1]);
  }

  return code;
}
