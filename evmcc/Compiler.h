
#pragma once

#include <libdevcore/Common.h>

namespace evmcc
{

class Compiler
{
public:

	Compiler();

	void compile(const dev::bytes& bytecode);

};

}