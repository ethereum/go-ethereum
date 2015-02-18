#ifndef ETHSERP_FUNCTIONS
#define ETHSERP_FUNCTIONS

#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"
#include "lllparser.h"
#include "bignum.h"
#include "optimize.h"
#include "rewriteutils.h"
#include "preprocess.h"


class argPack {
    public:
        argPack(Node a, Node b, Node c) {
            pre = a;
            datastart = b;
            datasz = c;
        }
    Node pre;
    Node datastart;
    Node datasz;
};

// Get a signature from a function
std::string getSignature(std::vector<Node> args);

// Convert a list of arguments into a <pre, mstart, msize> node
// triple, given the signature of a function
Node packArguments(std::vector<Node> args, std::string sig,
                   int funId, Metadata m);

// Create a node for argument unpacking
Node unpackArguments(std::vector<Node> vars, Metadata m);

#endif
