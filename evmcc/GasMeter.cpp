
#include "GasMeter.h"

#include <libevmface/Instruction.h>
#include <libevm/FeeStructure.h>

namespace evmcc
{

using namespace dev::eth; // We should move all the JIT code into dev::eth namespace

namespace
{

uint64_t getStepCost(dev::eth::Instruction inst) // TODO: Add this function to FeeSructure
{
	switch (inst)
	{
	case Instruction::STOP:
	case Instruction::SUICIDE:
		return 0;

	case Instruction::SSTORE:
		return static_cast<uint64_t>(c_sstoreGas);

	case Instruction::SLOAD:
		return static_cast<uint64_t>(c_sloadGas);

	case Instruction::SHA3:
		return static_cast<uint64_t>(c_sha3Gas);

	case Instruction::BALANCE:
		return static_cast<uint64_t>(c_sha3Gas);

	case Instruction::CALL:
	case Instruction::CALLCODE:
		return static_cast<uint64_t>(c_callGas);

	case Instruction::CREATE:
		return static_cast<uint64_t>(c_createGas);

	default: // Assumes instruction code is valid
		return static_cast<uint64_t>(c_stepGas);;
	}
}

}

}