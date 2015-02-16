#ifndef ETHSERP_OPTIMIZER
#define ETHSERP_OPTIMIZER

#include <stdio.h>
#include <iostream>
#include <vector>
#include <map>
#include "util.h"

// Compile-time arithmetic calculations
Node optimize(Node inp);

// Is a node degenerate (ie. trivial to calculate) ?
bool isDegenerate(Node n);

// Is a node purely arithmetic?
bool isPureArithmetic(Node n);

#endif
