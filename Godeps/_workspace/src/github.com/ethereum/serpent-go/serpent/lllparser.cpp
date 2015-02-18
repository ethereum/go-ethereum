#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"
#include "lllparser.h"
#include "tokenize.h"

struct _parseOutput {
    Node node;
    int newpos;
};

// Helper, returns subtree and position of start of next node
_parseOutput _parse(std::vector<Node> inp, int pos) {
    Metadata met = inp[pos].metadata;
    _parseOutput o;
    // Bracket: keep grabbing tokens until we get to the
    // corresponding closing bracket
    if (inp[pos].val == "(" || inp[pos].val == "[") {
        std::string fun, rbrack;
        std::vector<Node> args;
        pos += 1;
        if (inp[pos].val == "[") {
            fun = "access";
            rbrack = "]";
        }
        else rbrack = ")";
        // First argument is the function
        while (inp[pos].val != ")") {
            _parseOutput po = _parse(inp, pos);
            if (fun.length() == 0 && po.node.type == 1) {
                std::cerr << "Error: first arg must be function\n";
                fun = po.node.val;
            }
            else if (fun.length() == 0) {
                fun = po.node.val;
            }
            else {
                args.push_back(po.node);
            }
            pos = po.newpos;
        }
        o.newpos = pos + 1;
        o.node = astnode(fun, args, met);
    }
    // Normal token, return it and advance to next token
    else {
        o.newpos = pos + 1;
        o.node = token(inp[pos].val, met);
    }
    return o;
}

// stream of tokens -> lisp parse tree
Node parseLLLTokenStream(std::vector<Node> inp) {
    _parseOutput o = _parse(inp, 0);
    return o.node;
}

// Parses LLL
Node parseLLL(std::string s, bool allowFileRead) {
    std::string input = s;
    std::string file = "main";
    if (exists(s) && allowFileRead) {
        file = s;
        input = get_file_contents(s);
    }
    return parseLLLTokenStream(tokenize(s, Metadata(file, 0, 0), true));
}
