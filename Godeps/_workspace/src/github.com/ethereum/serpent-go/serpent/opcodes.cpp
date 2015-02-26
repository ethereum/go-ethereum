#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "opcodes.h"
#include "util.h"
#include "bignum.h"

Mapping mapping[] = {
    Mapping("STOP", 0x00, 0, 0),
    Mapping("ADD", 0x01, 2, 1),
    Mapping("MUL", 0x02, 2, 1),
    Mapping("SUB", 0x03, 2, 1),
    Mapping("DIV", 0x04, 2, 1),
    Mapping("SDIV", 0x05, 2, 1),
    Mapping("MOD", 0x06, 2, 1),
    Mapping("SMOD", 0x07, 2, 1),
    Mapping("ADDMOD", 0x08, 3, 1),
    Mapping("MULMOD", 0x09, 3, 1),
    Mapping("EXP", 0x0a, 2, 1),
    Mapping("SIGNEXTEND", 0x0b, 2, 1),
    Mapping("LT", 0x10, 2, 1),
    Mapping("GT", 0x11, 2, 1),
    Mapping("SLT", 0x12, 2, 1),
    Mapping("SGT", 0x13, 2, 1),
    Mapping("EQ", 0x14, 2, 1),
    Mapping("ISZERO", 0x15, 1, 1),
    Mapping("AND", 0x16, 2, 1),
    Mapping("OR", 0x17, 2, 1),
    Mapping("XOR", 0x18, 2, 1),
    Mapping("NOT", 0x19, 1, 1),
    Mapping("BYTE", 0x1a, 2, 1),
    Mapping("SHA3", 0x20, 2, 1),
    Mapping("ADDRESS", 0x30, 0, 1),
    Mapping("BALANCE", 0x31, 1, 1),
    Mapping("ORIGIN", 0x32, 0, 1),
    Mapping("CALLER", 0x33, 0, 1),
    Mapping("CALLVALUE", 0x34, 0, 1),
    Mapping("CALLDATALOAD", 0x35, 1, 1),
    Mapping("CALLDATASIZE", 0x36, 0, 1),
    Mapping("CALLDATACOPY", 0x37, 3, 0),
    Mapping("CODESIZE", 0x38, 0, 1),
    Mapping("CODECOPY", 0x39, 3, 0),
    Mapping("GASPRICE", 0x3a, 0, 1),
    Mapping("EXTCODESIZE", 0x3b, 1, 1),
    Mapping("EXTCODECOPY", 0x3c, 4, 0),
    Mapping("PREVHASH", 0x40, 0, 1),
    Mapping("COINBASE", 0x41, 0, 1),
    Mapping("TIMESTAMP", 0x42, 0, 1),
    Mapping("NUMBER", 0x43, 0, 1),
    Mapping("DIFFICULTY", 0x44, 0, 1),
    Mapping("GASLIMIT", 0x45, 0, 1),
    Mapping("POP", 0x50, 1, 0),
    Mapping("MLOAD", 0x51, 1, 1),
    Mapping("MSTORE", 0x52, 2, 0),
    Mapping("MSTORE8", 0x53, 2, 0),
    Mapping("SLOAD", 0x54, 1, 1),
    Mapping("SSTORE", 0x55, 2, 0),
    Mapping("JUMP", 0x56, 1, 0),
    Mapping("JUMPI", 0x57, 2, 0),
    Mapping("PC", 0x58, 0, 1),
    Mapping("MSIZE", 0x59, 0, 1),
    Mapping("GAS", 0x5a, 0, 1),
    Mapping("JUMPDEST", 0x5b, 0, 0),
    Mapping("LOG0", 0xa0, 2, 0),
    Mapping("LOG1", 0xa1, 3, 0),
    Mapping("LOG2", 0xa2, 4, 0),
    Mapping("LOG3", 0xa3, 5, 0),
    Mapping("LOG4", 0xa4, 6, 0),
    Mapping("CREATE", 0xf0, 3, 1),
    Mapping("CALL", 0xf1, 7, 1),
    Mapping("CALLCODE", 0xf2, 7, 1),
    Mapping("RETURN", 0xf3, 2, 0),
    Mapping("SUICIDE", 0xff, 1, 0),
    Mapping("---END---", 0x00, 0, 0),
};

std::map<std::string, std::vector<int> > opcodes;
std::map<int, std::string> reverseOpcodes;

// Fetches everything EXCEPT PUSH1..32
std::pair<std::string, std::vector<int> > _opdata(std::string ops, int opi) {
    if (!opcodes.size()) {
        int i = 0;
        while (mapping[i].op != "---END---") {
            Mapping mi = mapping[i];
            opcodes[mi.op] = triple(mi.opcode, mi.in, mi.out);
            i++;
        }
        for (i = 1; i <= 16; i++) {
            opcodes["DUP"+unsignedToDecimal(i)] = triple(0x7f + i, i, i+1);
            opcodes["SWAP"+unsignedToDecimal(i)] = triple(0x8f + i, i+1, i+1);
        }
        for (std::map<std::string, std::vector<int> >::iterator it=opcodes.begin();
             it != opcodes.end();
             it++) {
            reverseOpcodes[(*it).second[0]] = (*it).first;
        }
    }
    ops = upperCase(ops);
    std::string op;
    std::vector<int> opdata;
    op = reverseOpcodes.count(opi) ? reverseOpcodes[opi] : "";
    opdata = opcodes.count(ops) ? opcodes[ops] : triple(-1, -1, -1);
    return std::pair<std::string, std::vector<int> >(op, opdata);
}

int opcode(std::string op) {
	return _opdata(op, -1).second[0];
}

int opinputs(std::string op) {
	return _opdata(op, -1).second[1];
}

int opoutputs(std::string op) {
	return _opdata(op, -1).second[2];
}

std::string op(int opcode) {
	return _opdata("", opcode).first;
}

std::string lllSpecials[][3] = {
    { "ref", "1", "1" },
    { "get", "1", "1" },
    { "set", "2", "2" },
    { "with", "3", "3" },
    { "comment", "0", "2147483647" },
    { "ops", "0", "2147483647" },
    { "lll", "2", "2" },
    { "seq", "0", "2147483647" },
    { "if", "3", "3" },
    { "unless", "2", "2" },
    { "until", "2", "2" },
    { "alloc", "1", "1" },
    { "---END---", "0", "0" },
};

std::map<std::string, std::pair<int, int> > lllMap;

// Is a function name one of the valid functions above?
bool isValidLLLFunc(std::string f, int argc) {
    if (lllMap.size() == 0) {
        for (int i = 0; ; i++) {
            if (lllSpecials[i][0] == "---END---") break;
            lllMap[lllSpecials[i][0]] = std::pair<int, int>(
                dtu(lllSpecials[i][1]), dtu(lllSpecials[i][2]));
        }
    }
    return lllMap.count(f)
        && argc >= lllMap[f].first
        && argc <= lllMap[f].second;
}
