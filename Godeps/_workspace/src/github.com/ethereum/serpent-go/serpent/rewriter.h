#ifndef ETHSERP_REWRITER
#define ETHSERP_REWRITER

#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"

// Applies rewrite rules
Node rewrite(Node inp);

// Applies rewrite rules adding without wrapper
Node rewriteChunk(Node inp);

#endif
