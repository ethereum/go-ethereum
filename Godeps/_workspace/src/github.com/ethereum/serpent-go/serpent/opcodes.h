#ifndef ETHSERP_OPCODES
#define ETHSERP_OPCODES

#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"

class Mapping {
    public:
        Mapping(std::string Op, int Opcode, int In, int Out) {
            op = Op;
            opcode = Opcode;
            in = In;
            out = Out;
        }
        std::string op;
        int opcode;
        int in;
        int out;
};

extern Mapping mapping[];

extern std::map<std::string, std::vector<int> > opcodes;
extern std::map<int, std::string> reverseOpcodes;

std::pair<std::string, std::vector<int> > _opdata(std::string ops, int opi);

int opcode(std::string op);

int opinputs(std::string op);

int opoutputs(std::string op);

std::string op(int opcode);

extern std::string lllSpecials[][3];

extern std::map<std::string, std::pair<int, int> > lllMap;

bool isValidLLLFunc(std::string f, int argc);

#endif
