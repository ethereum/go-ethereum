
#include "GasMeter.h"

#include <llvm/IR/GlobalVariable.h>
#include <llvm/IR/Function.h>

#include <libevmface/Instruction.h>
#include <libevm/FeeStructure.h>

#include "Type.h"

namespace evmcc
{

using namespace dev::eth; // We should move all the JIT code into dev::eth namespace

namespace // Helper functions
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

GasMeter::GasMeter(llvm::IRBuilder<>& _builder, llvm::Module* _module):
	m_builder(_builder)
{
	m_gas = new llvm::GlobalVariable(*_module, Type::i256, false, llvm::GlobalVariable::PrivateLinkage, llvm::UndefValue::get(Type::i256), "gas");
	m_gas->setUnnamedAddr(true); // Address is not important

	//llvm::Function::Create()
}

void GasMeter::check(Instruction _inst)
{
	auto stepCost = getStepCost(_inst);
	auto before = m_builder.CreateLoad(m_gas, "gas.before");
	auto after = m_builder.CreateSub(before, m_builder.getIntN(256, stepCost));
	m_builder.CreateStore(after, m_gas);
}

}