#ifndef ETHSERP_PREPROCESSOR
#define ETHSERP_PREPROCESSOR

#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"

// Storage variable index storing object
struct svObj {
    std::map<std::string, std::string> offsets;
    std::map<std::string, int> indices;
    std::map<std::string, std::vector<std::string> > coefficients;
    std::map<std::string, bool> nonfinal;
    std::string globalOffset;
};

class rewriteRule {
    public:
        rewriteRule(Node p, Node s) {
            pattern = p;
            substitution = s;
        }
        Node pattern;
        Node substitution;
};


// Preprocessing result storing object
class preprocessAux {
    public:
        preprocessAux() {
            globalExterns = std::map<std::string, int>();
            localExterns = std::map<std::string, std::map<std::string, int> >();
            localExterns["self"] = std::map<std::string, int>();
        }
        std::map<std::string, int> globalExterns;
        std::map<std::string, std::string> globalExternSigs;
        std::map<std::string, std::map<std::string, int> > localExterns;
        std::map<std::string, std::map<std::string, std::string> > localExternSigs;
        std::vector<rewriteRule> customMacros;
        std::map<std::string, std::string> types;
        svObj storageVars;
};

#define preprocessResult std::pair<Node, preprocessAux>

// Populate an svObj with the arguments needed to determine
// the storage position of a node
svObj getStorageVars(svObj pre, Node node, std::string prefix="",
                     int index=0);

// Preprocess a function (see cpp for details)
preprocessResult preprocess(Node inp);


#endif
