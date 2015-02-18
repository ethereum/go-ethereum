#ifndef ETHSERP_COMPILER
#define ETHSERP_COMPILER

#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"

// Compiled fragtree -> compiled fragtree without labels
Node dereference(Node program);

// LLL -> fragtree
Node buildFragmentTree(Node program);

// Dereferenced fragtree -> opcodes
std::vector<Node> flatten(Node derefed);

// opcodes -> bin
std::string serialize(std::vector<Node> codons);

// Fragtree -> bin
std::string assemble(Node fragTree);

// Fragtree -> opcodes
std::vector<Node> prettyAssemble(Node fragTree);

// LLL -> bin
std::string compileLLL(Node program);

// LLL -> opcodes
std::vector<Node> prettyCompileLLL(Node program);

// bin -> opcodes
std::vector<Node> deserialize(std::string ser);

// Converts a list of integer values to binary transaction data
std::string encodeDatalist(std::vector<std::string> vals);

// Converts binary transaction data into a list of integer values
std::vector<std::string> decodeDatalist(std::string ser);

#endif
