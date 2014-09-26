
#pragma once

#include <libdevcore/Common.h>

namespace evmcc
{

class ExecutionEngine
{
public:
	ExecutionEngine();

	int run(const dev::bytes& bytecode);
};

}