
#pragma once

#include <llvm/IR/IRBuilder.h>

#include "Common.h"

namespace dev
{
namespace eth
{
namespace jit
{

struct JIT: public NoteChannel  { static const char* name() { return "JIT"; } };

#define clog(CHANNEL) std::cerr

/// Representation of 256-bit value binary compatible with LLVM i256
// TODO: Replace with h256
struct i256
{
	uint64_t a;
	uint64_t b;
	uint64_t c;
	uint64_t d;
};
static_assert(sizeof(i256) == 32, "Wrong i265 size");

u256 llvm2eth(i256);
i256 eth2llvm(u256);

}
}
}
