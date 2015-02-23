#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"
#include "lllparser.h"
#include "bignum.h"
#include "rewriteutils.h"
#include "optimize.h"
#include "preprocess.h"
#include "functions.h"
#include "opcodes.h"

// Convert a function of the form (def (f x y z) (do stuff)) into
// (if (first byte of ABI is correct) (seq (setup x y z) (do stuff)))
Node convFunction(Node node, int functionCount) {
    std::string prefix = "_temp"+mkUniqueToken()+"_";
    Metadata m = node.metadata;

    if (node.args.size() != 2)
        err("Malformed def!", m);
    // Collect the list of variable names and variable byte counts
    Node unpack = unpackArguments(node.args[0].args, m);
    // And the actual code
    Node body = node.args[1];
    // Main LLL-based function body
    return astnode("if",
                   astnode("eq",
                           astnode("get", token("__funid", m), m),
                           token(unsignedToDecimal(functionCount), m),
                           m),
                   astnode("seq", unpack, body, m));
}

// Populate an svObj with the arguments needed to determine
// the storage position of a node
svObj getStorageVars(svObj pre, Node node, std::string prefix,
                     int index) {
    Metadata m = node.metadata;
    if (!pre.globalOffset.size()) pre.globalOffset = "0";
    std::vector<Node> h;
    std::vector<std::string> coefficients;
    // Array accesses or atoms
    if (node.val == "access" || node.type == TOKEN) {
        std::string tot = "1";
        h = listfyStorageAccess(node);
        coefficients.push_back("1");
        for (unsigned i = h.size() - 1; i >= 1; i--) {
            // Array sizes must be constant or at least arithmetically
            // evaluable at compile time
            if (!isPureArithmetic(h[i]))
                err("Array size must be fixed value", m);
            // Create a list of the coefficient associated with each
            // array index
            coefficients.push_back(decimalMul(coefficients.back(), h[i].val));
        }
    }
    // Tuples
    else {
        int startc;
        // Handle the (fun <fun_astnode> args...) case
        if (node.val == "fun") {
            startc = 1;
            h = listfyStorageAccess(node.args[0]);
        }
        // Handle the (<fun_name> args...) case, which
        // the serpent parser produces when the function
        // is a simple name and not a complex astnode
        else {
            startc = 0;
            h = listfyStorageAccess(token(node.val, m));
        }
        svObj sub = pre;
        sub.globalOffset = "0";
        // Evaluate tuple elements recursively
        for (unsigned i = startc; i < node.args.size(); i++) {
            sub = getStorageVars(sub,
                                 node.args[i],
                                 prefix+h[0].val.substr(2)+".",
                                 i-startc);
        }
        coefficients.push_back(sub.globalOffset);
        for (unsigned i = h.size() - 1; i >= 1; i--) {
            // Array sizes must be constant or at least arithmetically
            // evaluable at compile time
            if (!isPureArithmetic(h[i]))
               err("Array size must be fixed value", m);
            // Create a list of the coefficient associated with each
            // array index
            coefficients.push_back(decimalMul(coefficients.back(), h[i].val));
        }
        pre.offsets = sub.offsets;
        pre.coefficients = sub.coefficients;
        pre.nonfinal = sub.nonfinal;
        pre.nonfinal[prefix+h[0].val.substr(2)] = true;
    }
    pre.coefficients[prefix+h[0].val.substr(2)] = coefficients;
    pre.offsets[prefix+h[0].val.substr(2)] = pre.globalOffset;
    pre.indices[prefix+h[0].val.substr(2)] = index;
    if (decimalGt(tt176, coefficients.back()))
        pre.globalOffset = decimalAdd(pre.globalOffset, coefficients.back());
    return pre;
}

// Preprocess input containing functions
//
// localExterns is a map of the form, eg,
//
// { x: { foo: 0, bar: 1, baz: 2 }, y: { qux: 0, foo: 1 } ... }
//
// localExternSigs is a map of the form, eg,
//
// { x : { foo: iii, bar: iis, baz: ia }, y: { qux: i, foo: as } ... }
//
// Signifying that x.foo = 0, x.baz = 2, y.foo = 1, etc
// and that x.foo has three integers as arguments, x.bar has two
// integers and a variable-length string, and baz has an integer
// and an array
//
// globalExterns is a one-level map, eg from above
//
// { foo: 1, bar: 1, baz: 2, qux: 0 }
//
// globalExternSigs is a one-level map, eg from above
//
// { foo: as, bar: iis, baz: ia, qux: i}
//
// Note that globalExterns and globalExternSigs may be ambiguous
// Also, a null signature implies an infinite tail of integers
preprocessResult preprocessInit(Node inp) {
    Metadata m = inp.metadata;
    if (inp.val != "seq")
        inp = astnode("seq", inp, m);
    std::vector<Node> empty = std::vector<Node>();
    Node init = astnode("seq", empty, m);
    Node shared = astnode("seq", empty, m);
    std::vector<Node> any;
    std::vector<Node> functions;
    preprocessAux out = preprocessAux();
    out.localExterns["self"] = std::map<std::string, int>();
    int functionCount = 0;
    int storageDataCount = 0;
    for (unsigned i = 0; i < inp.args.size(); i++) {
        Node obj = inp.args[i];
        // Functions
        if (obj.val == "def") {
            if (obj.args.size() == 0)
                err("Empty def", m);
            std::string funName = obj.args[0].val;
            // Init, shared and any are special functions
            if (funName == "init" || funName == "shared" || funName == "any") {
                if (obj.args[0].args.size())
                    err(funName+" cannot have arguments", m);
            }
            if (funName == "init") init = obj.args[1];
            else if (funName == "shared") shared = obj.args[1];
            else if (funName == "any") any.push_back(obj.args[1]);
            else {
                // Other functions
                functions.push_back(convFunction(obj, functionCount));
                out.localExterns["self"][obj.args[0].val] = functionCount;
                out.localExternSigs["self"][obj.args[0].val] 
                    = getSignature(obj.args[0].args);
                functionCount++;
            }
        }
        // Extern declarations
        else if (obj.val == "extern") {
            std::string externName = obj.args[0].val;
            Node al = obj.args[1];
            if (!out.localExterns.count(externName))
                out.localExterns[externName] = std::map<std::string, int>();
            for (unsigned i = 0; i < al.args.size(); i++) {
                if (al.args[i].val == ":") {
                    std::string v = al.args[i].args[0].val;
                    std::string sig = al.args[i].args[1].val;
                    out.globalExterns[v] = i;
                    out.globalExternSigs[v] = sig;
                    out.localExterns[externName][v] = i;
                    out.localExternSigs[externName][v] = sig;
                }
                else {
                    std::string v = al.args[i].val;
                    out.globalExterns[v] = i;
                    out.globalExternSigs[v] = "";
                    out.localExterns[externName][v] = i;
                    out.localExternSigs[externName][v] = "";
                }
            }
        }
        // Custom macros
        else if (obj.val == "macro") {
            // Rules for valid macros:
            //
            // There are only four categories of valid macros:
            //
            // 1. a macro where the outer function is something
            // which is NOT an existing valid function/extern/datum
            // 2. a macro of the form set(c(x), d) where c must NOT
            // be an existing valid function/extern/datum
            // 3. something of the form access(c(x)), where c must NOT
            // be an existing valid function/extern/datum
            // 4. something of the form set(access(c(x)), d) where c must
            // NOT be an existing valid function/extern/datum
            bool valid = false;
            Node pattern = obj.args[0];
            Node substitution = obj.args[1];
            if (opcode(pattern.val) < 0 && !isValidFunctionName(pattern.val))
                valid = true;
            if (pattern.val == "set" &&
                    opcode(pattern.args[0].val) < 0 &&
                    !isValidFunctionName(pattern.args[0].val))
                valid = true;
            if (pattern.val == "access" &&
                    opcode(pattern.args[0].val) < 0 &&
                    !isValidFunctionName(pattern.args[0].val))
            if (pattern.val == "set" &&
                    pattern.args[0].val == "access" &&
                    opcode(pattern.args[0].args[0].val) < 0 &&
                    !isValidFunctionName(pattern.args[0].args[0].val))
                valid = true;
            if (valid) {
                out.customMacros.push_back(rewriteRule(pattern, substitution));
            }
        }
        // Variable types
        else if (obj.val == "type") {
            std::string typeName = obj.args[0].val;
            std::vector<Node> vars = obj.args[1].args;
            for (unsigned i = 0; i < vars.size(); i++)
                out.types[vars[i].val] = typeName;
        }
        // Storage variables/structures
        else if (obj.val == "data") {
            out.storageVars = getStorageVars(out.storageVars,
                                             obj.args[0],
                                             "",
                                             storageDataCount);
            storageDataCount += 1;
        }
        else any.push_back(obj);
    }
    std::vector<Node> main;
    if (shared.args.size()) main.push_back(shared);
    if (init.args.size()) main.push_back(init);

    std::vector<Node> code;
    if (shared.args.size()) code.push_back(shared);
    for (unsigned i = 0; i < any.size(); i++)
        code.push_back(any[i]);
    for (unsigned i = 0; i < functions.size(); i++)
        code.push_back(functions[i]);
    Node codeNode;
    if (functions.size() > 0) {
        codeNode = astnode("with",
                           token("__funid", m),
                           astnode("byte",
                                   token("0", m),
                                   astnode("calldataload", token("0", m), m),
                                   m),
                           astnode("seq", code, m),
                           m);
    }
    else codeNode = astnode("seq", code, m);
    main.push_back(astnode("~return",
                           token("0", m),
                           astnode("lll",
                                   codeNode,
                                   token("0", m),
                                   m),
                           m));


    Node result;
    if (main.size() == 1) result = main[0];
    else result = astnode("seq", main, inp.metadata);
    return preprocessResult(result, out);
}

preprocessResult processTypes (preprocessResult pr) {
    preprocessAux aux = pr.second;
    Node node = pr.first;
    if (node.type == TOKEN && aux.types.count(node.val)) {
        node = asn(aux.types[node.val], node, node.metadata);
    }
    else if (node.val == "untyped")
        return preprocessResult(node.args[0], aux);
    else {
        for (unsigned i = 0; i < node.args.size(); i++) {
            node.args[i] =
                processTypes(preprocessResult(node.args[i], aux)).first;
        }
    }
    return preprocessResult(node, aux);
}

preprocessResult preprocess(Node n) {
    return processTypes(preprocessInit(n));
}
