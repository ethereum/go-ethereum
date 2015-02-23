#ifndef ETHSERP_TOKENIZE
#define ETHSERP_TOKENIZE

#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"

int chartype(char c);

std::vector<Node> tokenize(std::string inp,
                           Metadata meta=Metadata(),
                           bool lispMode=false);

#endif
