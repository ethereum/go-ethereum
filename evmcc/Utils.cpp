
#include "Utils.h"

namespace evmcc
{

dev::u256 llvm2eth(i256 _i)
{
	dev::u256 u = 0;
	u |= _i.d;
	u <<= 64;
	u |= _i.c;
	u <<= 64;
	u |= _i.b;
	u <<= 64;
	u |= _i.a;
	return u;
}

i256 eth2llvm(dev::u256 _u)
{
	i256 i;
	dev::u256 mask = 0xFFFFFFFFFFFFFFFF;
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