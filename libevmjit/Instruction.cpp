
#include "Instruction.h"
#include <llvm/ADT/APInt.h>

namespace dev
{
namespace eth
{
namespace jit
{

llvm::APInt readPushData(bytes::const_iterator& _curr, bytes::const_iterator _end)
{
	auto pushInst = *_curr;
	assert(Instruction(pushInst) >= Instruction::PUSH1 && Instruction(pushInst) <= Instruction::PUSH32);
	auto numBytes = pushInst - static_cast<size_t>(Instruction::PUSH1) + 1;
	llvm::APInt value(256, 0);
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

void skipPushData(bytes::const_iterator& _curr, bytes::const_iterator _end)
{
	auto pushInst = *_curr;
	assert(Instruction(pushInst) >= Instruction::PUSH1 && Instruction(pushInst) <= Instruction::PUSH32);
	auto numBytes = pushInst - static_cast<size_t>(Instruction::PUSH1) + 1;
	--_end;
	for (decltype(numBytes) i = 0; i < numBytes && _curr < _end; ++i, ++_curr) {}
}

}
}
}
