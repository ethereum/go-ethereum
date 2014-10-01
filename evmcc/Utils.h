
#pragma once

#include <cstdint>

#include <libdevcore/Common.h>

namespace evmcc
{

/// Representation of 256-bit value binary compatible with LLVM i256
struct i256
{
	uint64_t a;
	uint64_t b;
	uint64_t c;
	uint64_t d;
};
static_assert(sizeof(i256) == 32, "Wrong i265 size");

dev::u256 llvm2eth(i256);
i256 eth2llvm(dev::u256);

}