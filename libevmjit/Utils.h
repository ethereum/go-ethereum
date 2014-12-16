#pragma once

#include "Common.h"

namespace dev
{
namespace eth
{
namespace jit
{

struct JIT: public NoteChannel  { static const char* name() { return "JIT"; } };

//#define clog(CHANNEL) std::cerr
#define clog(CHANNEL) std::ostream(nullptr)

u256 llvm2eth(i256);
i256 eth2llvm(u256);

void terminate(ReturnCode _returnCode);

}
}
}
