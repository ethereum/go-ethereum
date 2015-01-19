#pragma once

#include <evmjit/libevmjit/Common.h>

namespace dev
{
namespace eth
{

inline u256 llvm2eth(jit::i256 _i)
{
	u256 u = 0;
	u |= _i.d;
	u <<= 64;
	u |= _i.c;
	u <<= 64;
	u |= _i.b;
	u <<= 64;
	u |= _i.a;
	return u;
}

inline jit::i256 eth2llvm(u256 _u)
{
	jit::i256 i;
	u256 mask = 0xFFFFFFFFFFFFFFFF;
	i.a = static_cast<uint64_t>(_u & mask);
	_u >>= 64;
	i.b = static_cast<uint64_t>(_u & mask);
	_u >>= 64;
	i.c = static_cast<uint64_t>(_u & mask);
	_u >>= 64;
	i.d = static_cast<uint64_t>(_u & mask);
	return i;
}

}
}
