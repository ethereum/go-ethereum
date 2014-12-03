
#pragma once

#include <csetjmp>

#include "Utils.h"


namespace dev
{
namespace eth
{
namespace jit
{
	
struct RuntimeData
{
	enum Index
	{
		Gas,
		Address,
		Caller,
		Origin,
		CallValue,
		CallDataSize,
		GasPrice,
		PrevHash,
		CoinBase,
		TimeStamp,
		Number,
		Difficulty,
		GasLimit,
		CodeSize,

		_size,

		ReturnDataOffset = CallValue,	// Reuse 2 fields for return data reference
		ReturnDataSize = CallDataSize
	};

	i256 elems[_size];
	byte const* callData;
	byte const* code;
};

}
}
}
