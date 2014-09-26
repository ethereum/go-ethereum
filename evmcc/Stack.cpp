
#include <vector>
#include <iostream>
#include <iomanip>
#include <cstdint>

#ifdef _MSC_VER
	#define EXPORT __declspec(dllexport)
#else
	#define EXPORT
#endif

struct i256
{
	uint64_t a;
	uint64_t b;
	uint64_t c;
	uint64_t d;
};

using Stack = std::vector<i256>;

extern "C"
{

EXPORT void* evmccrt_stack_create()
{
	std::cerr << "STACK create: ";
	auto stack = new Stack;
	std::cerr << stack << "\n";
	return stack;
}

EXPORT void evmccrt_stack_push(void* _stack, uint64_t _partA, uint64_t _partB, uint64_t _partC, uint64_t _partD)
{
	std::cerr << "STACK push: " << _partA << " (" << std::hex << std::setfill('0')
			  << std::setw(16) << _partD << " " 
			  << std::setw(16) << _partC << " " 
			  << std::setw(16) << _partB << " "
			  << std::setw(16) << _partA;
	auto stack = static_cast<Stack*>(_stack);
	stack->push_back({_partA, _partB, _partC, _partD});
	std::cerr << ")\n";
}

}	// extern "C"
