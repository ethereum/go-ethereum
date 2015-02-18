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
#include "functions.h"

std::string getSignature(std::vector<Node> args) {
    std::string o;
    for (unsigned i = 0; i < args.size(); i++) {
        if (args[i].val == ":" && args[i].args[1].val == "s")
            o += "s";
        else if (args[i].val == ":" && args[i].args[1].val == "a")
            o += "a";
        else
            o += "i";
    }
    return o;
}

// Convert a list of arguments into a node containing a
// < datastart, datasz > pair

Node packArguments(std::vector<Node> args, std::string sig,
                      int funId, Metadata m) {
    // Plain old 32 byte arguments
    std::vector<Node> nargs;
    // Variable-sized arguments
    std::vector<Node> vargs;
    // Variable sizes
    std::vector<Node> sizes;
    // Is a variable an array?
    std::vector<bool> isArray;
    // Fill up above three argument lists
    int argCount = 0;
    for (unsigned i = 0; i < args.size(); i++) {
        Metadata m = args[i].metadata;
        if (args[i].val == "=") {
            // do nothing
        }
        else {
            // Determine the correct argument type
            char argType;
            if (sig.size() > 0) {
                if (argCount >= (signed)sig.size())
                    err("Too many args", m);
                argType = sig[argCount];
            }
            else argType = 'i';
            // Integer (also usable for short strings)
            if (argType == 'i') {
                if (args[i].val == ":")
                    err("Function asks for int, provided string or array", m);
                nargs.push_back(args[i]);
            }
            // Long string
            else if (argType == 's') {
                if (args[i].val != ":")
                    err("Must specify string length", m);
                vargs.push_back(args[i].args[0]);
                sizes.push_back(args[i].args[1]);
                isArray.push_back(false);
            }
            // Array
            else if (argType == 'a') {
                if (args[i].val != ":")
                    err("Must specify array length", m);
                vargs.push_back(args[i].args[0]);
                sizes.push_back(args[i].args[1]);
                isArray.push_back(true);
            }
            else err("Invalid arg type in signature", m);
            argCount++;
        }
    }
    int static_arg_size = 1 + (vargs.size() + nargs.size()) * 32;
    // Start off by saving the size variables and calculating the total
    msn kwargs;
    kwargs["funid"] = tkn(utd(funId), m);
    std::string pattern =
        "(with _sztot "+utd(static_arg_size)+"                            "
        "    (with _sizes (alloc "+utd(sizes.size() * 32)+")              "
        "        (seq                                                     ";
    for (unsigned i = 0; i < sizes.size(); i++) {
        std::string sizeIncrement = 
            isArray[i] ? "(mul 32 _x)" : "_x";
        pattern +=
            "(with _x $sz"+utd(i)+"(seq                                   "
            "    (mstore (add _sizes "+utd(i * 32)+") _x)                 "
            "    (set _sztot (add _sztot "+sizeIncrement+" ))))           ";
        kwargs["sz"+utd(i)] = sizes[i];
    }
    // Allocate memory, and set first data byte
    pattern +=
            "(with _datastart (alloc (add _sztot 32)) (seq                "
            "    (mstore8 _datastart $funid)                              ";
    // Copy over size variables
    for (unsigned i = 0; i < sizes.size(); i++) {
        int v = 1 + i * 32;
        pattern +=
            "    (mstore                                                  "
            "          (add _datastart "+utd(v)+")                        "
            "          (mload (add _sizes "+utd(v-1)+")))                 ";
    }
    // Store normal arguments
    for (unsigned i = 0; i < nargs.size(); i++) {
        int v = 1 + (i + sizes.size()) * 32;
        pattern +=
            "    (mstore (add _datastart "+utd(v)+") $"+utd(i)+")         ";
        kwargs[utd(i)] = nargs[i];
    }
    // Loop through variable-sized arguments, store them
    pattern += 
            "    (with _pos (add _datastart "+utd(static_arg_size)+") (seq";
    for (unsigned i = 0; i < vargs.size(); i++) {
        std::string copySize =
            isArray[i] ? "(mul 32 (mload (add _sizes "+utd(i * 32)+")))"
                       : "(mload (add _sizes "+utd(i * 32)+"))";
        pattern +=
            "        (unsafe_mcopy _pos $vl"+utd(i)+" "+copySize+")       "
            "        (set _pos (add _pos "+copySize+"))                   ";
        kwargs["vl"+utd(i)] = vargs[i];
    }
    // Return a 2-item array containing the start and size
    pattern += "     (array_lit _datastart _sztot))))))))";
    std::string prefix = "_temp_"+mkUniqueToken();
    // Fill in pattern, return triple
    return subst(parseLLL(pattern), kwargs, prefix, m);
}

// Create a node for argument unpacking
Node unpackArguments(std::vector<Node> vars, Metadata m) {
    std::vector<std::string> varNames;
    std::vector<std::string> longVarNames;
    std::vector<bool> longVarIsArray;
    // Fill in variable and long variable names, as well as which
    // long variables are arrays and which are strings
    for (unsigned i = 0; i < vars.size(); i++) {
        if (vars[i].val == ":") {
            if (vars[i].args.size() != 2)
                err("Malformed def!", m);
            longVarNames.push_back(vars[i].args[0].val);
            std::string tag = vars[i].args[1].val;
            if (tag == "s")
                longVarIsArray.push_back(false);
            else if (tag == "a")
                longVarIsArray.push_back(true);
            else
                err("Function value can only be string or array", m);
        }
        else {
            varNames.push_back(vars[i].val);
        }
    }
    std::vector<Node> sub;
    if (!varNames.size() && !longVarNames.size()) {
        // do nothing if we have no arguments
    }
    else {
        std::vector<Node> varNodes;
        for (unsigned i = 0; i < longVarNames.size(); i++)
            varNodes.push_back(token(longVarNames[i], m));
        for (unsigned i = 0; i < varNames.size(); i++)
            varNodes.push_back(token(varNames[i], m));
        // Copy over variable lengths and short variables
        for (unsigned i = 0; i < varNodes.size(); i++) {
            int pos = 1 + i * 32;
            std::string prefix = (i < longVarNames.size()) ? "_len_" : "";
            sub.push_back(asn("untyped", asn("set",
                              token(prefix+varNodes[i].val, m),
                              asn("calldataload", tkn(utd(pos), m), m),
                              m)));
        }
        // Copy over long variables
        if (longVarNames.size() > 0) {
            std::vector<Node> sub2;
            int pos = varNodes.size() * 32 + 1;
            Node tot = tkn("_tot", m);
            for (unsigned i = 0; i < longVarNames.size(); i++) {
                Node var = tkn(longVarNames[i], m);
                Node varlen = longVarIsArray[i] 
                    ? asn("mul", tkn("32", m), tkn("_len_"+longVarNames[i], m))
                    : tkn("_len_"+longVarNames[i], m);
                sub2.push_back(asn("untyped",
                                   asn("set", var, asn("alloc", varlen))));
                sub2.push_back(asn("calldatacopy", var, tot, varlen));
                sub2.push_back(asn("set", tot, asn("add", tot, varlen)));
            }
            std::string prefix = "_temp_"+mkUniqueToken();
            sub.push_back(subst(
                astnode("with", tot, tkn(utd(pos), m), asn("seq", sub2)),
                msn(),
                prefix,
                m));
        }
    }
    return asn("seq", sub, m);
}
