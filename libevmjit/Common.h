#pragma once

#include <vector>
#include <boost/multiprecision/cpp_int.hpp>

namespace dev
{
namespace eth
{
namespace jit
{

using byte = uint8_t;
using bytes = std::vector<byte>;
using bytes_ref = std::tuple<byte const*, size_t>;
using u256 = boost::multiprecision::uint256_t;
using bigint = boost::multiprecision::cpp_int;

struct NoteChannel {};	// FIXME: Use some log library?

enum class ReturnCode
{
	Stop = 0,
	Return = 1,
	Suicide = 2,

	BadJumpDestination = 101,
	OutOfGas = 102,
	StackTooSmall = 103,
	BadInstruction = 104,

	LLVMConfigError = 201,
	LLVMCompileError = 202,
	LLVMLinkError = 203,
};

/// Representation of 256-bit value binary compatible with LLVM i256
struct i256
{
	uint64_t a = 0;
	uint64_t b = 0;
	uint64_t c = 0;
	uint64_t d = 0;
};
static_assert(sizeof(i256) == 32, "Wrong i265 size");

#define UNTESTED assert(false)

}
}
}
