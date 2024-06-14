"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isCreate = exports.isCall = exports.getOpcodeLength = exports.getPushLength = exports.isJump = exports.isPush = exports.opcodeName = exports.Opcode = void 0;
var Opcode;
(function (Opcode) {
    // Arithmetic operations
    Opcode[Opcode["STOP"] = 0] = "STOP";
    Opcode[Opcode["ADD"] = 1] = "ADD";
    Opcode[Opcode["MUL"] = 2] = "MUL";
    Opcode[Opcode["SUB"] = 3] = "SUB";
    Opcode[Opcode["DIV"] = 4] = "DIV";
    Opcode[Opcode["SDIV"] = 5] = "SDIV";
    Opcode[Opcode["MOD"] = 6] = "MOD";
    Opcode[Opcode["SMOD"] = 7] = "SMOD";
    Opcode[Opcode["ADDMOD"] = 8] = "ADDMOD";
    Opcode[Opcode["MULMOD"] = 9] = "MULMOD";
    Opcode[Opcode["EXP"] = 10] = "EXP";
    Opcode[Opcode["SIGNEXTEND"] = 11] = "SIGNEXTEND";
    // Unallocated
    Opcode[Opcode["UNRECOGNIZED_0C"] = 12] = "UNRECOGNIZED_0C";
    Opcode[Opcode["UNRECOGNIZED_0D"] = 13] = "UNRECOGNIZED_0D";
    Opcode[Opcode["UNRECOGNIZED_0E"] = 14] = "UNRECOGNIZED_0E";
    Opcode[Opcode["UNRECOGNIZED_0F"] = 15] = "UNRECOGNIZED_0F";
    // Comparison and bitwise operations
    Opcode[Opcode["LT"] = 16] = "LT";
    Opcode[Opcode["GT"] = 17] = "GT";
    Opcode[Opcode["SLT"] = 18] = "SLT";
    Opcode[Opcode["SGT"] = 19] = "SGT";
    Opcode[Opcode["EQ"] = 20] = "EQ";
    Opcode[Opcode["ISZERO"] = 21] = "ISZERO";
    Opcode[Opcode["AND"] = 22] = "AND";
    Opcode[Opcode["OR"] = 23] = "OR";
    Opcode[Opcode["XOR"] = 24] = "XOR";
    Opcode[Opcode["NOT"] = 25] = "NOT";
    Opcode[Opcode["BYTE"] = 26] = "BYTE";
    Opcode[Opcode["SHL"] = 27] = "SHL";
    Opcode[Opcode["SHR"] = 28] = "SHR";
    Opcode[Opcode["SAR"] = 29] = "SAR";
    // Unallocated
    Opcode[Opcode["UNRECOGNIZED_1E"] = 30] = "UNRECOGNIZED_1E";
    Opcode[Opcode["UNRECOGNIZED_1F"] = 31] = "UNRECOGNIZED_1F";
    // Cryptographic operations
    Opcode[Opcode["SHA3"] = 32] = "SHA3";
    // Unallocated
    Opcode[Opcode["UNRECOGNIZED_21"] = 33] = "UNRECOGNIZED_21";
    Opcode[Opcode["UNRECOGNIZED_22"] = 34] = "UNRECOGNIZED_22";
    Opcode[Opcode["UNRECOGNIZED_23"] = 35] = "UNRECOGNIZED_23";
    Opcode[Opcode["UNRECOGNIZED_24"] = 36] = "UNRECOGNIZED_24";
    Opcode[Opcode["UNRECOGNIZED_25"] = 37] = "UNRECOGNIZED_25";
    Opcode[Opcode["UNRECOGNIZED_26"] = 38] = "UNRECOGNIZED_26";
    Opcode[Opcode["UNRECOGNIZED_27"] = 39] = "UNRECOGNIZED_27";
    Opcode[Opcode["UNRECOGNIZED_28"] = 40] = "UNRECOGNIZED_28";
    Opcode[Opcode["UNRECOGNIZED_29"] = 41] = "UNRECOGNIZED_29";
    Opcode[Opcode["UNRECOGNIZED_2A"] = 42] = "UNRECOGNIZED_2A";
    Opcode[Opcode["UNRECOGNIZED_2B"] = 43] = "UNRECOGNIZED_2B";
    Opcode[Opcode["UNRECOGNIZED_2C"] = 44] = "UNRECOGNIZED_2C";
    Opcode[Opcode["UNRECOGNIZED_2D"] = 45] = "UNRECOGNIZED_2D";
    Opcode[Opcode["UNRECOGNIZED_2E"] = 46] = "UNRECOGNIZED_2E";
    Opcode[Opcode["UNRECOGNIZED_2F"] = 47] = "UNRECOGNIZED_2F";
    // Message info operations
    Opcode[Opcode["ADDRESS"] = 48] = "ADDRESS";
    Opcode[Opcode["BALANCE"] = 49] = "BALANCE";
    Opcode[Opcode["ORIGIN"] = 50] = "ORIGIN";
    Opcode[Opcode["CALLER"] = 51] = "CALLER";
    Opcode[Opcode["CALLVALUE"] = 52] = "CALLVALUE";
    Opcode[Opcode["CALLDATALOAD"] = 53] = "CALLDATALOAD";
    Opcode[Opcode["CALLDATASIZE"] = 54] = "CALLDATASIZE";
    Opcode[Opcode["CALLDATACOPY"] = 55] = "CALLDATACOPY";
    Opcode[Opcode["CODESIZE"] = 56] = "CODESIZE";
    Opcode[Opcode["CODECOPY"] = 57] = "CODECOPY";
    Opcode[Opcode["GASPRICE"] = 58] = "GASPRICE";
    Opcode[Opcode["EXTCODESIZE"] = 59] = "EXTCODESIZE";
    Opcode[Opcode["EXTCODECOPY"] = 60] = "EXTCODECOPY";
    Opcode[Opcode["RETURNDATASIZE"] = 61] = "RETURNDATASIZE";
    Opcode[Opcode["RETURNDATACOPY"] = 62] = "RETURNDATACOPY";
    Opcode[Opcode["EXTCODEHASH"] = 63] = "EXTCODEHASH";
    // Block info operations
    Opcode[Opcode["BLOCKHASH"] = 64] = "BLOCKHASH";
    Opcode[Opcode["COINBASE"] = 65] = "COINBASE";
    Opcode[Opcode["TIMESTAMP"] = 66] = "TIMESTAMP";
    Opcode[Opcode["NUMBER"] = 67] = "NUMBER";
    Opcode[Opcode["DIFFICULTY"] = 68] = "DIFFICULTY";
    Opcode[Opcode["GASLIMIT"] = 69] = "GASLIMIT";
    // Istanbul opcodes
    Opcode[Opcode["CHAINID"] = 70] = "CHAINID";
    Opcode[Opcode["SELFBALANCE"] = 71] = "SELFBALANCE";
    // London opcodes
    Opcode[Opcode["BASEFEE"] = 72] = "BASEFEE";
    // Unallocated
    Opcode[Opcode["UNRECOGNIZED_49"] = 73] = "UNRECOGNIZED_49";
    Opcode[Opcode["UNRECOGNIZED_4A"] = 74] = "UNRECOGNIZED_4A";
    Opcode[Opcode["UNRECOGNIZED_4B"] = 75] = "UNRECOGNIZED_4B";
    Opcode[Opcode["UNRECOGNIZED_4C"] = 76] = "UNRECOGNIZED_4C";
    Opcode[Opcode["UNRECOGNIZED_4D"] = 77] = "UNRECOGNIZED_4D";
    Opcode[Opcode["UNRECOGNIZED_4E"] = 78] = "UNRECOGNIZED_4E";
    Opcode[Opcode["UNRECOGNIZED_4F"] = 79] = "UNRECOGNIZED_4F";
    // Storage, memory, and other operations
    Opcode[Opcode["POP"] = 80] = "POP";
    Opcode[Opcode["MLOAD"] = 81] = "MLOAD";
    Opcode[Opcode["MSTORE"] = 82] = "MSTORE";
    Opcode[Opcode["MSTORE8"] = 83] = "MSTORE8";
    Opcode[Opcode["SLOAD"] = 84] = "SLOAD";
    Opcode[Opcode["SSTORE"] = 85] = "SSTORE";
    Opcode[Opcode["JUMP"] = 86] = "JUMP";
    Opcode[Opcode["JUMPI"] = 87] = "JUMPI";
    Opcode[Opcode["PC"] = 88] = "PC";
    Opcode[Opcode["MSIZE"] = 89] = "MSIZE";
    Opcode[Opcode["GAS"] = 90] = "GAS";
    Opcode[Opcode["JUMPDEST"] = 91] = "JUMPDEST";
    // Uncallocated
    Opcode[Opcode["UNRECOGNIZED_5C"] = 92] = "UNRECOGNIZED_5C";
    Opcode[Opcode["UNRECOGNIZED_5D"] = 93] = "UNRECOGNIZED_5D";
    Opcode[Opcode["UNRECOGNIZED_5E"] = 94] = "UNRECOGNIZED_5E";
    Opcode[Opcode["UNRECOGNIZED_5F"] = 95] = "UNRECOGNIZED_5F";
    // Push operations
    Opcode[Opcode["PUSH1"] = 96] = "PUSH1";
    Opcode[Opcode["PUSH2"] = 97] = "PUSH2";
    Opcode[Opcode["PUSH3"] = 98] = "PUSH3";
    Opcode[Opcode["PUSH4"] = 99] = "PUSH4";
    Opcode[Opcode["PUSH5"] = 100] = "PUSH5";
    Opcode[Opcode["PUSH6"] = 101] = "PUSH6";
    Opcode[Opcode["PUSH7"] = 102] = "PUSH7";
    Opcode[Opcode["PUSH8"] = 103] = "PUSH8";
    Opcode[Opcode["PUSH9"] = 104] = "PUSH9";
    Opcode[Opcode["PUSH10"] = 105] = "PUSH10";
    Opcode[Opcode["PUSH11"] = 106] = "PUSH11";
    Opcode[Opcode["PUSH12"] = 107] = "PUSH12";
    Opcode[Opcode["PUSH13"] = 108] = "PUSH13";
    Opcode[Opcode["PUSH14"] = 109] = "PUSH14";
    Opcode[Opcode["PUSH15"] = 110] = "PUSH15";
    Opcode[Opcode["PUSH16"] = 111] = "PUSH16";
    Opcode[Opcode["PUSH17"] = 112] = "PUSH17";
    Opcode[Opcode["PUSH18"] = 113] = "PUSH18";
    Opcode[Opcode["PUSH19"] = 114] = "PUSH19";
    Opcode[Opcode["PUSH20"] = 115] = "PUSH20";
    Opcode[Opcode["PUSH21"] = 116] = "PUSH21";
    Opcode[Opcode["PUSH22"] = 117] = "PUSH22";
    Opcode[Opcode["PUSH23"] = 118] = "PUSH23";
    Opcode[Opcode["PUSH24"] = 119] = "PUSH24";
    Opcode[Opcode["PUSH25"] = 120] = "PUSH25";
    Opcode[Opcode["PUSH26"] = 121] = "PUSH26";
    Opcode[Opcode["PUSH27"] = 122] = "PUSH27";
    Opcode[Opcode["PUSH28"] = 123] = "PUSH28";
    Opcode[Opcode["PUSH29"] = 124] = "PUSH29";
    Opcode[Opcode["PUSH30"] = 125] = "PUSH30";
    Opcode[Opcode["PUSH31"] = 126] = "PUSH31";
    Opcode[Opcode["PUSH32"] = 127] = "PUSH32";
    // Dup operations
    Opcode[Opcode["DUP1"] = 128] = "DUP1";
    Opcode[Opcode["DUP2"] = 129] = "DUP2";
    Opcode[Opcode["DUP3"] = 130] = "DUP3";
    Opcode[Opcode["DUP4"] = 131] = "DUP4";
    Opcode[Opcode["DUP5"] = 132] = "DUP5";
    Opcode[Opcode["DUP6"] = 133] = "DUP6";
    Opcode[Opcode["DUP7"] = 134] = "DUP7";
    Opcode[Opcode["DUP8"] = 135] = "DUP8";
    Opcode[Opcode["DUP9"] = 136] = "DUP9";
    Opcode[Opcode["DUP10"] = 137] = "DUP10";
    Opcode[Opcode["DUP11"] = 138] = "DUP11";
    Opcode[Opcode["DUP12"] = 139] = "DUP12";
    Opcode[Opcode["DUP13"] = 140] = "DUP13";
    Opcode[Opcode["DUP14"] = 141] = "DUP14";
    Opcode[Opcode["DUP15"] = 142] = "DUP15";
    Opcode[Opcode["DUP16"] = 143] = "DUP16";
    // Swap operations
    Opcode[Opcode["SWAP1"] = 144] = "SWAP1";
    Opcode[Opcode["SWAP2"] = 145] = "SWAP2";
    Opcode[Opcode["SWAP3"] = 146] = "SWAP3";
    Opcode[Opcode["SWAP4"] = 147] = "SWAP4";
    Opcode[Opcode["SWAP5"] = 148] = "SWAP5";
    Opcode[Opcode["SWAP6"] = 149] = "SWAP6";
    Opcode[Opcode["SWAP7"] = 150] = "SWAP7";
    Opcode[Opcode["SWAP8"] = 151] = "SWAP8";
    Opcode[Opcode["SWAP9"] = 152] = "SWAP9";
    Opcode[Opcode["SWAP10"] = 153] = "SWAP10";
    Opcode[Opcode["SWAP11"] = 154] = "SWAP11";
    Opcode[Opcode["SWAP12"] = 155] = "SWAP12";
    Opcode[Opcode["SWAP13"] = 156] = "SWAP13";
    Opcode[Opcode["SWAP14"] = 157] = "SWAP14";
    Opcode[Opcode["SWAP15"] = 158] = "SWAP15";
    Opcode[Opcode["SWAP16"] = 159] = "SWAP16";
    // Log operations
    Opcode[Opcode["LOG0"] = 160] = "LOG0";
    Opcode[Opcode["LOG1"] = 161] = "LOG1";
    Opcode[Opcode["LOG2"] = 162] = "LOG2";
    Opcode[Opcode["LOG3"] = 163] = "LOG3";
    Opcode[Opcode["LOG4"] = 164] = "LOG4";
    // Unallocated
    Opcode[Opcode["UNRECOGNIZED_A5"] = 165] = "UNRECOGNIZED_A5";
    Opcode[Opcode["UNRECOGNIZED_A6"] = 166] = "UNRECOGNIZED_A6";
    Opcode[Opcode["UNRECOGNIZED_A7"] = 167] = "UNRECOGNIZED_A7";
    Opcode[Opcode["UNRECOGNIZED_A8"] = 168] = "UNRECOGNIZED_A8";
    Opcode[Opcode["UNRECOGNIZED_A9"] = 169] = "UNRECOGNIZED_A9";
    Opcode[Opcode["UNRECOGNIZED_AA"] = 170] = "UNRECOGNIZED_AA";
    Opcode[Opcode["UNRECOGNIZED_AB"] = 171] = "UNRECOGNIZED_AB";
    Opcode[Opcode["UNRECOGNIZED_AC"] = 172] = "UNRECOGNIZED_AC";
    Opcode[Opcode["UNRECOGNIZED_AD"] = 173] = "UNRECOGNIZED_AD";
    Opcode[Opcode["UNRECOGNIZED_AE"] = 174] = "UNRECOGNIZED_AE";
    Opcode[Opcode["UNRECOGNIZED_AF"] = 175] = "UNRECOGNIZED_AF";
    Opcode[Opcode["UNRECOGNIZED_B0"] = 176] = "UNRECOGNIZED_B0";
    Opcode[Opcode["UNRECOGNIZED_B1"] = 177] = "UNRECOGNIZED_B1";
    Opcode[Opcode["UNRECOGNIZED_B2"] = 178] = "UNRECOGNIZED_B2";
    Opcode[Opcode["UNRECOGNIZED_B3"] = 179] = "UNRECOGNIZED_B3";
    Opcode[Opcode["UNRECOGNIZED_B4"] = 180] = "UNRECOGNIZED_B4";
    Opcode[Opcode["UNRECOGNIZED_B5"] = 181] = "UNRECOGNIZED_B5";
    Opcode[Opcode["UNRECOGNIZED_B6"] = 182] = "UNRECOGNIZED_B6";
    Opcode[Opcode["UNRECOGNIZED_B7"] = 183] = "UNRECOGNIZED_B7";
    Opcode[Opcode["UNRECOGNIZED_B8"] = 184] = "UNRECOGNIZED_B8";
    Opcode[Opcode["UNRECOGNIZED_B9"] = 185] = "UNRECOGNIZED_B9";
    Opcode[Opcode["UNRECOGNIZED_BA"] = 186] = "UNRECOGNIZED_BA";
    Opcode[Opcode["UNRECOGNIZED_BB"] = 187] = "UNRECOGNIZED_BB";
    Opcode[Opcode["UNRECOGNIZED_BC"] = 188] = "UNRECOGNIZED_BC";
    Opcode[Opcode["UNRECOGNIZED_BD"] = 189] = "UNRECOGNIZED_BD";
    Opcode[Opcode["UNRECOGNIZED_BE"] = 190] = "UNRECOGNIZED_BE";
    Opcode[Opcode["UNRECOGNIZED_BF"] = 191] = "UNRECOGNIZED_BF";
    Opcode[Opcode["UNRECOGNIZED_C0"] = 192] = "UNRECOGNIZED_C0";
    Opcode[Opcode["UNRECOGNIZED_C1"] = 193] = "UNRECOGNIZED_C1";
    Opcode[Opcode["UNRECOGNIZED_C2"] = 194] = "UNRECOGNIZED_C2";
    Opcode[Opcode["UNRECOGNIZED_C3"] = 195] = "UNRECOGNIZED_C3";
    Opcode[Opcode["UNRECOGNIZED_C4"] = 196] = "UNRECOGNIZED_C4";
    Opcode[Opcode["UNRECOGNIZED_C5"] = 197] = "UNRECOGNIZED_C5";
    Opcode[Opcode["UNRECOGNIZED_C6"] = 198] = "UNRECOGNIZED_C6";
    Opcode[Opcode["UNRECOGNIZED_C7"] = 199] = "UNRECOGNIZED_C7";
    Opcode[Opcode["UNRECOGNIZED_C8"] = 200] = "UNRECOGNIZED_C8";
    Opcode[Opcode["UNRECOGNIZED_C9"] = 201] = "UNRECOGNIZED_C9";
    Opcode[Opcode["UNRECOGNIZED_CA"] = 202] = "UNRECOGNIZED_CA";
    Opcode[Opcode["UNRECOGNIZED_CB"] = 203] = "UNRECOGNIZED_CB";
    Opcode[Opcode["UNRECOGNIZED_CC"] = 204] = "UNRECOGNIZED_CC";
    Opcode[Opcode["UNRECOGNIZED_CD"] = 205] = "UNRECOGNIZED_CD";
    Opcode[Opcode["UNRECOGNIZED_CE"] = 206] = "UNRECOGNIZED_CE";
    Opcode[Opcode["UNRECOGNIZED_CF"] = 207] = "UNRECOGNIZED_CF";
    Opcode[Opcode["UNRECOGNIZED_D0"] = 208] = "UNRECOGNIZED_D0";
    Opcode[Opcode["UNRECOGNIZED_D1"] = 209] = "UNRECOGNIZED_D1";
    Opcode[Opcode["UNRECOGNIZED_D2"] = 210] = "UNRECOGNIZED_D2";
    Opcode[Opcode["UNRECOGNIZED_D3"] = 211] = "UNRECOGNIZED_D3";
    Opcode[Opcode["UNRECOGNIZED_D4"] = 212] = "UNRECOGNIZED_D4";
    Opcode[Opcode["UNRECOGNIZED_D5"] = 213] = "UNRECOGNIZED_D5";
    Opcode[Opcode["UNRECOGNIZED_D6"] = 214] = "UNRECOGNIZED_D6";
    Opcode[Opcode["UNRECOGNIZED_D7"] = 215] = "UNRECOGNIZED_D7";
    Opcode[Opcode["UNRECOGNIZED_D8"] = 216] = "UNRECOGNIZED_D8";
    Opcode[Opcode["UNRECOGNIZED_D9"] = 217] = "UNRECOGNIZED_D9";
    Opcode[Opcode["UNRECOGNIZED_DA"] = 218] = "UNRECOGNIZED_DA";
    Opcode[Opcode["UNRECOGNIZED_DB"] = 219] = "UNRECOGNIZED_DB";
    Opcode[Opcode["UNRECOGNIZED_DC"] = 220] = "UNRECOGNIZED_DC";
    Opcode[Opcode["UNRECOGNIZED_DD"] = 221] = "UNRECOGNIZED_DD";
    Opcode[Opcode["UNRECOGNIZED_DE"] = 222] = "UNRECOGNIZED_DE";
    Opcode[Opcode["UNRECOGNIZED_DF"] = 223] = "UNRECOGNIZED_DF";
    Opcode[Opcode["UNRECOGNIZED_E0"] = 224] = "UNRECOGNIZED_E0";
    Opcode[Opcode["UNRECOGNIZED_E1"] = 225] = "UNRECOGNIZED_E1";
    Opcode[Opcode["UNRECOGNIZED_E2"] = 226] = "UNRECOGNIZED_E2";
    Opcode[Opcode["UNRECOGNIZED_E3"] = 227] = "UNRECOGNIZED_E3";
    Opcode[Opcode["UNRECOGNIZED_E4"] = 228] = "UNRECOGNIZED_E4";
    Opcode[Opcode["UNRECOGNIZED_E5"] = 229] = "UNRECOGNIZED_E5";
    Opcode[Opcode["UNRECOGNIZED_E6"] = 230] = "UNRECOGNIZED_E6";
    Opcode[Opcode["UNRECOGNIZED_E7"] = 231] = "UNRECOGNIZED_E7";
    Opcode[Opcode["UNRECOGNIZED_E8"] = 232] = "UNRECOGNIZED_E8";
    Opcode[Opcode["UNRECOGNIZED_E9"] = 233] = "UNRECOGNIZED_E9";
    Opcode[Opcode["UNRECOGNIZED_EA"] = 234] = "UNRECOGNIZED_EA";
    Opcode[Opcode["UNRECOGNIZED_EB"] = 235] = "UNRECOGNIZED_EB";
    Opcode[Opcode["UNRECOGNIZED_EC"] = 236] = "UNRECOGNIZED_EC";
    Opcode[Opcode["UNRECOGNIZED_ED"] = 237] = "UNRECOGNIZED_ED";
    Opcode[Opcode["UNRECOGNIZED_EE"] = 238] = "UNRECOGNIZED_EE";
    Opcode[Opcode["UNRECOGNIZED_EF"] = 239] = "UNRECOGNIZED_EF";
    // Call operations
    Opcode[Opcode["CREATE"] = 240] = "CREATE";
    Opcode[Opcode["CALL"] = 241] = "CALL";
    Opcode[Opcode["CALLCODE"] = 242] = "CALLCODE";
    Opcode[Opcode["RETURN"] = 243] = "RETURN";
    Opcode[Opcode["DELEGATECALL"] = 244] = "DELEGATECALL";
    Opcode[Opcode["CREATE2"] = 245] = "CREATE2";
    // Unallocated
    Opcode[Opcode["UNRECOGNIZED_F6"] = 246] = "UNRECOGNIZED_F6";
    Opcode[Opcode["UNRECOGNIZED_F7"] = 247] = "UNRECOGNIZED_F7";
    Opcode[Opcode["UNRECOGNIZED_F8"] = 248] = "UNRECOGNIZED_F8";
    Opcode[Opcode["UNRECOGNIZED_F9"] = 249] = "UNRECOGNIZED_F9";
    // Other operations
    Opcode[Opcode["STATICCALL"] = 250] = "STATICCALL";
    // Unallocated
    Opcode[Opcode["UNRECOGNIZED_FB"] = 251] = "UNRECOGNIZED_FB";
    Opcode[Opcode["UNRECOGNIZED_FC"] = 252] = "UNRECOGNIZED_FC";
    // Other operations
    Opcode[Opcode["REVERT"] = 253] = "REVERT";
    Opcode[Opcode["INVALID"] = 254] = "INVALID";
    Opcode[Opcode["SELFDESTRUCT"] = 255] = "SELFDESTRUCT";
})(Opcode = exports.Opcode || (exports.Opcode = {}));
function opcodeName(opcode) {
    return Opcode[opcode] ?? `<unrecognized opcode ${opcode}>`;
}
exports.opcodeName = opcodeName;
function isPush(opcode) {
    return opcode >= Opcode.PUSH1 && opcode <= Opcode.PUSH32;
}
exports.isPush = isPush;
function isJump(opcode) {
    return opcode === Opcode.JUMP || opcode === Opcode.JUMPI;
}
exports.isJump = isJump;
function getPushLength(opcode) {
    return opcode - Opcode.PUSH1 + 1;
}
exports.getPushLength = getPushLength;
function getOpcodeLength(opcode) {
    if (!isPush(opcode)) {
        return 1;
    }
    return 1 + getPushLength(opcode);
}
exports.getOpcodeLength = getOpcodeLength;
function isCall(opcode) {
    return (opcode === Opcode.CALL ||
        opcode === Opcode.CALLCODE ||
        opcode === Opcode.DELEGATECALL ||
        opcode === Opcode.STATICCALL);
}
exports.isCall = isCall;
function isCreate(opcode) {
    return opcode === Opcode.CREATE || opcode === Opcode.CREATE2;
}
exports.isCreate = isCreate;
//# sourceMappingURL=opcodes.js.map