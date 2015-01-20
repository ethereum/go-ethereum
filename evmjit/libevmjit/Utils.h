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

}
}
}
