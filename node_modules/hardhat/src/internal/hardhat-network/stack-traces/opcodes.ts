export enum Opcode {
  // Arithmetic operations
  STOP = 0x00,
  ADD = 0x01,
  MUL = 0x02,
  SUB = 0x03,
  DIV = 0x04,
  SDIV = 0x05,
  MOD = 0x06,
  SMOD = 0x07,
  ADDMOD = 0x08,
  MULMOD = 0x09,
  EXP = 0x0a,
  SIGNEXTEND = 0x0b,

  // Unallocated
  UNRECOGNIZED_0C = 0x0c,
  UNRECOGNIZED_0D = 0x0d,
  UNRECOGNIZED_0E = 0x0e,
  UNRECOGNIZED_0F = 0x0f,

  // Comparison and bitwise operations
  LT = 0x10,
  GT = 0x11,
  SLT = 0x12,
  SGT = 0x13,
  EQ = 0x14,
  ISZERO = 0x15,
  AND = 0x16,
  OR = 0x17,
  XOR = 0x18,
  NOT = 0x19,
  BYTE = 0x1a,
  SHL = 0x1b,
  SHR = 0x1c,
  SAR = 0x1d,

  // Unallocated
  UNRECOGNIZED_1E = 0x1e,
  UNRECOGNIZED_1F = 0x1f,

  // Cryptographic operations
  SHA3 = 0x20,

  // Unallocated
  UNRECOGNIZED_21 = 0x21,
  UNRECOGNIZED_22 = 0x22,
  UNRECOGNIZED_23 = 0x23,
  UNRECOGNIZED_24 = 0x24,
  UNRECOGNIZED_25 = 0x25,
  UNRECOGNIZED_26 = 0x26,
  UNRECOGNIZED_27 = 0x27,
  UNRECOGNIZED_28 = 0x28,
  UNRECOGNIZED_29 = 0x29,
  UNRECOGNIZED_2A = 0x2a,
  UNRECOGNIZED_2B = 0x2b,
  UNRECOGNIZED_2C = 0x2c,
  UNRECOGNIZED_2D = 0x2d,
  UNRECOGNIZED_2E = 0x2e,
  UNRECOGNIZED_2F = 0x2f,

  // Message info operations
  ADDRESS = 0x30,
  BALANCE = 0x31,
  ORIGIN = 0x32,
  CALLER = 0x33,
  CALLVALUE = 0x34,
  CALLDATALOAD = 0x35,
  CALLDATASIZE = 0x36,
  CALLDATACOPY = 0x37,
  CODESIZE = 0x38,
  CODECOPY = 0x39,
  GASPRICE = 0x3a,
  EXTCODESIZE = 0x3b,
  EXTCODECOPY = 0x3c,
  RETURNDATASIZE = 0x3d,
  RETURNDATACOPY = 0x3e,
  EXTCODEHASH = 0x3f,

  // Block info operations
  BLOCKHASH = 0x40,
  COINBASE = 0x41,
  TIMESTAMP = 0x42,
  NUMBER = 0x43,
  DIFFICULTY = 0x44,
  GASLIMIT = 0x45,

  // Istanbul opcodes
  CHAINID = 0x46,
  SELFBALANCE = 0x47,

  // London opcodes
  BASEFEE = 0x48,

  // Unallocated
  UNRECOGNIZED_49 = 0x49,
  UNRECOGNIZED_4A = 0x4a,
  UNRECOGNIZED_4B = 0x4b,
  UNRECOGNIZED_4C = 0x4c,
  UNRECOGNIZED_4D = 0x4d,
  UNRECOGNIZED_4E = 0x4e,
  UNRECOGNIZED_4F = 0x4f,

  // Storage, memory, and other operations
  POP = 0x50,
  MLOAD = 0x51,
  MSTORE = 0x52,
  MSTORE8 = 0x53,
  SLOAD = 0x54,
  SSTORE = 0x55,
  JUMP = 0x56,
  JUMPI = 0x57,
  PC = 0x58,
  MSIZE = 0x59,
  GAS = 0x5a,
  JUMPDEST = 0x5b,

  // Uncallocated
  UNRECOGNIZED_5C = 0x5c,
  UNRECOGNIZED_5D = 0x5d,
  UNRECOGNIZED_5E = 0x5e,
  UNRECOGNIZED_5F = 0x5f,

  // Push operations
  PUSH1 = 0x60,
  PUSH2 = 0x61,
  PUSH3 = 0x62,
  PUSH4 = 0x63,
  PUSH5 = 0x64,
  PUSH6 = 0x65,
  PUSH7 = 0x66,
  PUSH8 = 0x67,
  PUSH9 = 0x68,
  PUSH10 = 0x69,
  PUSH11 = 0x6a,
  PUSH12 = 0x6b,
  PUSH13 = 0x6c,
  PUSH14 = 0x6d,
  PUSH15 = 0x6e,
  PUSH16 = 0x6f,
  PUSH17 = 0x70,
  PUSH18 = 0x71,
  PUSH19 = 0x72,
  PUSH20 = 0x73,
  PUSH21 = 0x74,
  PUSH22 = 0x75,
  PUSH23 = 0x76,
  PUSH24 = 0x77,
  PUSH25 = 0x78,
  PUSH26 = 0x79,
  PUSH27 = 0x7a,
  PUSH28 = 0x7b,
  PUSH29 = 0x7c,
  PUSH30 = 0x7d,
  PUSH31 = 0x7e,
  PUSH32 = 0x7f,

  // Dup operations
  DUP1 = 0x80,
  DUP2 = 0x81,
  DUP3 = 0x82,
  DUP4 = 0x83,
  DUP5 = 0x84,
  DUP6 = 0x85,
  DUP7 = 0x86,
  DUP8 = 0x87,
  DUP9 = 0x88,
  DUP10 = 0x89,
  DUP11 = 0x8a,
  DUP12 = 0x8b,
  DUP13 = 0x8c,
  DUP14 = 0x8d,
  DUP15 = 0x8e,
  DUP16 = 0x8f,

  // Swap operations
  SWAP1 = 0x90,
  SWAP2 = 0x91,
  SWAP3 = 0x92,
  SWAP4 = 0x93,
  SWAP5 = 0x94,
  SWAP6 = 0x95,
  SWAP7 = 0x96,
  SWAP8 = 0x97,
  SWAP9 = 0x98,
  SWAP10 = 0x99,
  SWAP11 = 0x9a,
  SWAP12 = 0x9b,
  SWAP13 = 0x9c,
  SWAP14 = 0x9d,
  SWAP15 = 0x9e,
  SWAP16 = 0x9f,

  // Log operations
  LOG0 = 0xa0,
  LOG1 = 0xa1,
  LOG2 = 0xa2,
  LOG3 = 0xa3,
  LOG4 = 0xa4,

  // Unallocated
  UNRECOGNIZED_A5 = 0xa5,
  UNRECOGNIZED_A6 = 0xa6,
  UNRECOGNIZED_A7 = 0xa7,
  UNRECOGNIZED_A8 = 0xa8,
  UNRECOGNIZED_A9 = 0xa9,
  UNRECOGNIZED_AA = 0xaa,
  UNRECOGNIZED_AB = 0xab,
  UNRECOGNIZED_AC = 0xac,
  UNRECOGNIZED_AD = 0xad,
  UNRECOGNIZED_AE = 0xae,
  UNRECOGNIZED_AF = 0xaf,

  UNRECOGNIZED_B0 = 0xb0,
  UNRECOGNIZED_B1 = 0xb1,
  UNRECOGNIZED_B2 = 0xb2,
  UNRECOGNIZED_B3 = 0xb3,
  UNRECOGNIZED_B4 = 0xb4,
  UNRECOGNIZED_B5 = 0xb5,
  UNRECOGNIZED_B6 = 0xb6,
  UNRECOGNIZED_B7 = 0xb7,
  UNRECOGNIZED_B8 = 0xb8,
  UNRECOGNIZED_B9 = 0xb9,
  UNRECOGNIZED_BA = 0xba,
  UNRECOGNIZED_BB = 0xbb,
  UNRECOGNIZED_BC = 0xbc,
  UNRECOGNIZED_BD = 0xbd,
  UNRECOGNIZED_BE = 0xbe,
  UNRECOGNIZED_BF = 0xbf,

  UNRECOGNIZED_C0 = 0xc0,
  UNRECOGNIZED_C1 = 0xc1,
  UNRECOGNIZED_C2 = 0xc2,
  UNRECOGNIZED_C3 = 0xc3,
  UNRECOGNIZED_C4 = 0xc4,
  UNRECOGNIZED_C5 = 0xc5,
  UNRECOGNIZED_C6 = 0xc6,
  UNRECOGNIZED_C7 = 0xc7,
  UNRECOGNIZED_C8 = 0xc8,
  UNRECOGNIZED_C9 = 0xc9,
  UNRECOGNIZED_CA = 0xca,
  UNRECOGNIZED_CB = 0xcb,
  UNRECOGNIZED_CC = 0xcc,
  UNRECOGNIZED_CD = 0xcd,
  UNRECOGNIZED_CE = 0xce,
  UNRECOGNIZED_CF = 0xcf,

  UNRECOGNIZED_D0 = 0xd0,
  UNRECOGNIZED_D1 = 0xd1,
  UNRECOGNIZED_D2 = 0xd2,
  UNRECOGNIZED_D3 = 0xd3,
  UNRECOGNIZED_D4 = 0xd4,
  UNRECOGNIZED_D5 = 0xd5,
  UNRECOGNIZED_D6 = 0xd6,
  UNRECOGNIZED_D7 = 0xd7,
  UNRECOGNIZED_D8 = 0xd8,
  UNRECOGNIZED_D9 = 0xd9,
  UNRECOGNIZED_DA = 0xda,
  UNRECOGNIZED_DB = 0xdb,
  UNRECOGNIZED_DC = 0xdc,
  UNRECOGNIZED_DD = 0xdd,
  UNRECOGNIZED_DE = 0xde,
  UNRECOGNIZED_DF = 0xdf,

  UNRECOGNIZED_E0 = 0xe0,
  UNRECOGNIZED_E1 = 0xe1,
  UNRECOGNIZED_E2 = 0xe2,
  UNRECOGNIZED_E3 = 0xe3,
  UNRECOGNIZED_E4 = 0xe4,
  UNRECOGNIZED_E5 = 0xe5,
  UNRECOGNIZED_E6 = 0xe6,
  UNRECOGNIZED_E7 = 0xe7,
  UNRECOGNIZED_E8 = 0xe8,
  UNRECOGNIZED_E9 = 0xe9,
  UNRECOGNIZED_EA = 0xea,
  UNRECOGNIZED_EB = 0xeb,
  UNRECOGNIZED_EC = 0xec,
  UNRECOGNIZED_ED = 0xed,
  UNRECOGNIZED_EE = 0xee,
  UNRECOGNIZED_EF = 0xef,

  // Call operations
  CREATE = 0xf0,
  CALL = 0xf1,
  CALLCODE = 0xf2,
  RETURN = 0xf3,
  DELEGATECALL = 0xf4,
  CREATE2 = 0xf5,

  // Unallocated
  UNRECOGNIZED_F6 = 0xf6,
  UNRECOGNIZED_F7 = 0xf7,
  UNRECOGNIZED_F8 = 0xf8,
  UNRECOGNIZED_F9 = 0xf9,

  // Other operations
  STATICCALL = 0xfa,

  // Unallocated
  UNRECOGNIZED_FB = 0xfb,
  UNRECOGNIZED_FC = 0xfc,

  // Other operations
  REVERT = 0xfd,
  INVALID = 0xfe,
  SELFDESTRUCT = 0xff,
}

export function opcodeName(opcode: number): string {
  return Opcode[opcode] ?? `<unrecognized opcode ${opcode}>`;
}

export function isPush(opcode: Opcode) {
  return opcode >= Opcode.PUSH1 && opcode <= Opcode.PUSH32;
}

export function isJump(opcode: Opcode) {
  return opcode === Opcode.JUMP || opcode === Opcode.JUMPI;
}

export function getPushLength(opcode: Opcode) {
  return opcode - Opcode.PUSH1 + 1;
}

export function getOpcodeLength(opcode: Opcode) {
  if (!isPush(opcode)) {
    return 1;
  }

  return 1 + getPushLength(opcode);
}

export function isCall(opcode: Opcode) {
  return (
    opcode === Opcode.CALL ||
    opcode === Opcode.CALLCODE ||
    opcode === Opcode.DELEGATECALL ||
    opcode === Opcode.STATICCALL
  );
}

export function isCreate(opcode: Opcode) {
  return opcode === Opcode.CREATE || opcode === Opcode.CREATE2;
}
