#ifndef ETHSERP_REWRITEUTILS
#define ETHSERP_REWRITEUTILS

#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"

// Valid functions and their min and max argument counts
extern std::string validFunctions[][3];

extern std::map<std::string, bool> vfMap;

bool isValidFunctionName(std::string f);

// Converts deep array access into ordered list of the arguments
// along the descent
std::vector<Node> listfyStorageAccess(Node node);

// Cool function for debug purposes (named cerrStringList to make
// all prints searchable via 'cerr')
void cerrStringList(std::vector<std::string> s, std::string suffix="");

// Is the given node something of the form
// self.cow
// self.horse[0]
// self.a[6][7][self.storage[3]].chicken[9]
bool isNodeStorageVariable(Node node);

// Applies rewrite rules adding without wrapper
Node rewriteChunk(Node inp);

// Match result storing object
struct matchResult {
    bool success;
    std::map<std::string, Node> map;
};

// Match node to pattern
matchResult match(Node p, Node n);

// Substitute node using pattern
Node subst(Node pattern,
           std::map<std::string, Node> dict,
           std::string varflag,
           Metadata m);

Node withTransform(Node source);

#endif
