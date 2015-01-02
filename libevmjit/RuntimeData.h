#pragma once

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
		ReturnDataSize = CallDataSize,
		SuicideDestAddress = Address,	///< Suicide balance destination address
	};

	i256 elems[_size] = {};
	byte const* callData = nullptr;
	byte const* code = nullptr;

	void set(Index _index, u256 _value) { elems[_index] = eth2llvm(_value); }
	u256 get(Index _index) { return llvm2eth(elems[_index]); }
};

/// VM Environment (ExtVM) opaque type
struct Env;

}
}
}
