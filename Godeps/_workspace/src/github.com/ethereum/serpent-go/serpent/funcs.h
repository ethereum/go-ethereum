#include <stdio.h>
#include <iostream>
#include <vector>
#include "bignum.h"
#include "util.h"
#include "parser.h"
#include "lllparser.h"
#include "compiler.h"
#include "rewriter.h"
#include "tokenize.h"

// Function listing:
//
// parseSerpent      (serpent -> AST)      std::string -> Node
// parseLLL          (LLL -> AST)          std::string -> Node
// rewrite           (apply rewrite rules) Node -> Node
// compileToLLL      (serpent -> LLL)      std::string -> Node
// compileLLL        (LLL -> EVMhex)       Node -> std::string
// prettyCompileLLL  (LLL -> EVMasm)       Node -> std::vector<Node>
// prettyCompile     (serpent -> EVMasm)   std::string -> std::vector>Node>
// compile           (serpent -> EVMhex)   std::string -> std::string
// get_file_contents (filename -> file)    std::string -> std::string
// exists            (does file exist?)    std::string -> bool

Node compileToLLL(std::string input);

Node compileChunkToLLL(std::string input);

std::string compile(std::string input);

std::vector<Node> prettyCompile(std::string input);

std::string compileChunk(std::string input);

std::vector<Node> prettyCompileChunk(std::string input);
