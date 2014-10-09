
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

	auto pt = m_builder.GetInsertPoint();
	auto bb = m_builder.GetInsertBlock();
	m_gasCheckFunc = llvm::Function::Create(llvm::FunctionType::get(Type::Void, Type::i256, false), llvm::Function::PrivateLinkage, "gas.check", _module);
	auto gasCheckBB = llvm::BasicBlock::Create(_builder.getContext(), {}, m_gasCheckFunc);
	m_builder.SetInsertPoint(gasCheckBB);
	llvm::Value* cost = m_gasCheckFunc->arg_begin();
	llvm::Value* gas = m_builder.CreateLoad(m_gas);
	gas = m_builder.CreateSub(gas, cost);
	m_builder.CreateStore(gas, m_gas);
	m_builder.CreateRetVoid();
	m_builder.SetInsertPoint(bb, pt);
}

void GasMeter::check(Instruction _inst)
{
	auto stepCost = getStepCost(_inst);
	m_builder.CreateCall(m_gasCheckFunc, m_builder.getIntN(256, stepCost));
}

}