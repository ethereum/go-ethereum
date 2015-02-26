#include <stdio.h>
#include <string>
#include <iostream>
#include <vector>
#include <map>
#include "funcs.h"

int main(int argv, char** argc) {
    if (argv == 1) {
        std::cerr << "Must provide a command and arguments! Try parse, rewrite, compile, assemble\n";
        return 0;
    }
    if (argv == 2 && std::string(argc[1]) == "--help" || std::string(argc[1]) == "-h" ) {
        std::cout << argc[1] << "\n";
        
        std::cout << "serpent command input\n";
        std::cout << "where input -s for from stdin, a file, or interpreted as serpent code if does not exist as file.";
        std::cout << "where command: \n";
        std::cout << " parse:          Just parses and returns s-expression code.\n";
        std::cout << " rewrite:        Parse, use rewrite rules print s-expressions of result.\n";
        std::cout << " compile:        Return resulting compiled EVM code in hex.\n";
        std::cout << " assemble:       Return result from step before compilation.\n";
        return 0;
    }
        
    std::string flag = "";
    std::string command = argc[1];
    std::string input;
    std::string secondInput;
    if (std::string(argc[1]) == "-s") {
        flag = command.substr(1);
        command = argc[2];
        input = "";
        std::string line;
        while (std::getline(std::cin, line)) {
            input += line + "\n";
        }
        secondInput = argv == 3 ? "" : argc[3];
    }
    else {
        if (argv == 2) {
            std::cerr << "Not enough arguments for serpent cmdline\n";
            throw(0);
        }
        input = argc[2];
        secondInput = argv == 3 ? "" : argc[3];
    }
    bool haveSec = secondInput.length() > 0;
    if (command == "parse" || command == "parse_serpent") {
        std::cout << printAST(parseSerpent(input), haveSec) << "\n";
    }
    else if (command == "rewrite") {
        std::cout << printAST(rewrite(parseLLL(input, true)), haveSec) << "\n";
    }
    else if (command == "compile_to_lll") {
        std::cout << printAST(compileToLLL(input), haveSec) << "\n";
    }
    else if (command == "rewrite_chunk") {
        std::cout << printAST(rewriteChunk(parseLLL(input, true)), haveSec) << "\n";
    }
    else if (command == "compile_chunk_to_lll") {
        std::cout << printAST(compileChunkToLLL(input), haveSec) << "\n";
    }
    else if (command == "build_fragtree") {
        std::cout << printAST(buildFragmentTree(parseLLL(input, true))) << "\n";
    }
    else if (command == "compile_lll") {
        std::cout << binToHex(compileLLL(parseLLL(input, true))) << "\n";
    }
    else if (command == "dereference") {
        std::cout << printAST(dereference(parseLLL(input, true)), haveSec) <<"\n";
    }
    else if (command == "pretty_assemble") {
        std::cout << printTokens(prettyAssemble(parseLLL(input, true))) <<"\n";
    }
    else if (command == "pretty_compile_lll") {
        std::cout << printTokens(prettyCompileLLL(parseLLL(input, true))) << "\n";
    }
    else if (command == "pretty_compile") {
        std::cout << printTokens(prettyCompile(input)) << "\n";
    }
    else if (command == "pretty_compile_chunk") {
        std::cout << printTokens(prettyCompileChunk(input)) << "\n";
    }
    else if (command == "assemble") {
        std::cout << assemble(parseLLL(input, true)) << "\n";
    }
    else if (command == "serialize") {
        std::cout << binToHex(serialize(tokenize(input, Metadata(), false))) << "\n";
    }
    else if (command == "flatten") {
        std::cout << printTokens(flatten(parseLLL(input, true))) << "\n";
    }
    else if (command == "deserialize") {
        std::cout << printTokens(deserialize(hexToBin(input))) << "\n";
    }
    else if (command == "compile") {
        std::cout << binToHex(compile(input)) << "\n";
    }
    else if (command == "compile_chunk") {
        std::cout << binToHex(compileChunk(input)) << "\n";
    }
    else if (command == "encode_datalist") {
        std::vector<Node> tokens = tokenize(input);
        std::vector<std::string> o;
        for (int i = 0; i < (int)tokens.size(); i++) {
            o.push_back(tokens[i].val);
        }
        std::cout << binToHex(encodeDatalist(o)) << "\n";
    }
    else if (command == "decode_datalist") {
        std::vector<std::string> o = decodeDatalist(hexToBin(input));
        std::vector<Node> tokens;
        for (int i = 0; i < (int)o.size(); i++)
            tokens.push_back(token(o[i]));
        std::cout << printTokens(tokens) << "\n";
    }
    else if (command == "tokenize") {
        std::cout << printTokens(tokenize(input));
    }
    else if (command == "biject") {
        if (argv == 3)
             std::cerr << "Not enough arguments for biject\n";
        int pos = decimalToUnsigned(secondInput);
        std::vector<Node> n = prettyCompile(input);
        if (pos >= (int)n.size())
             std::cerr << "Code position too high\n";
        Metadata m = n[pos].metadata;
        std::cout << "Opcode: " << n[pos].val << ", file: " << m.file << 
             ", line: " << m.ln << ", char: " << m.ch << "\n";
    }
}
