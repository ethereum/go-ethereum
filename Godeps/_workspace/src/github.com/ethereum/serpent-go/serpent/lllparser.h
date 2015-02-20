#ifndef ETHSERP_LLLPARSER
#define ETHSERP_LLLPARSER

#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"

// LLL text -> parse tree
Node parseLLL(std::string s, bool allowFileRead=false);

#endif
