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
using u256 = boost::multiprecision::uint256_t;
using bigint = boost::multiprecision::cpp_int;

struct NoteChannel {};	// FIXME: Use some log library?

}
}
}
