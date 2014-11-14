
#include "Utils.h"

namespace dev
{
namespace eth
{
namespace jit
{

u256 llvm2eth(i256 _i)
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

i256 eth2llvm(u256 _u)
{
	i256 i;
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

u256 readPushData(bytes::const_iterator& _curr, bytes::const_iterator _end)
{
	auto pushInst = *_curr;
	assert(Instruction(pushInst) >= Instruction::PUSH1 && Instruction(pushInst) <= Instruction::PUSH32);
	auto numBytes = pushInst - static_cast<size_t>(Instruction::PUSH1) + 1;
	u256 value;
	++_curr;	// Point the data
	for (decltype(numBytes) i = 0; i < numBytes; ++i)
	{
		byte b = (_curr != _end) ? *_curr++ : 0;
		value <<= 8;
		value |= b;
	}
	--_curr;	// Point the last real byte read
	return value;
}

}
}
}
