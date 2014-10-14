
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
		return static_cast<uint64_t>(c_stepGas);
	}
}

bool isCostBlockEnd(Instruction _inst)
{
	// Basic block terminators like STOP are not needed on the list
	// as cost will be commited at the end of basic block

	// CALL & CALLCODE are commited manually

	switch (_inst)
	{
	case Instruction::CALLDATACOPY:
	case Instruction::CODECOPY:
	case Instruction::MLOAD:
	case Instruction::MSTORE:
	case Instruction::MSTORE8:
	case Instruction::SSTORE:
	case Instruction::GAS:
	case Instruction::CREATE:
		return true;

	default:
		return false;
	}
}

}

GasMeter::GasMeter(llvm::IRBuilder<>& _builder, llvm::Module* _module):
	m_builder(_builder)
{
	m_gas = new llvm::GlobalVariable(*_module, Type::i256, false, llvm::GlobalVariable::ExternalLinkage, nullptr, "gas");
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

void GasMeter::count(Instruction _inst)
{
	if (!m_checkCall)
	{
		// Create gas check call with mocked block cost at begining of current cost-block
		m_checkCall = m_builder.CreateCall(m_gasCheckFunc, llvm::UndefValue::get(Type::i256));
	}
	
	m_blockCost += getStepCost(_inst);

	if (isCostBlockEnd(_inst))
		commitCostBlock();
}

void GasMeter::giveBack(llvm::Value* _gas)
{
	llvm::Value* gasCounter = m_builder.CreateLoad(m_gas, "gas");
	gasCounter = m_builder.CreateAdd(gasCounter, _gas);
	m_builder.CreateStore(gasCounter, m_gas);
}

void GasMeter::commitCostBlock(llvm::Value* _additionalCost)
{
	assert(!_additionalCost || m_checkCall); // _additionalCost => m_checkCall; Must be inside cost-block

	// If any uncommited block
	if (m_checkCall)
	{
		if (m_blockCost == 0 && !_additionalCost) // Do not check 0
		{
			m_checkCall->eraseFromParent(); // Remove the gas check call
			return;
		}

		llvm::Value* cost = Constant::get(m_blockCost);
		if (_additionalCost)
			cost = m_builder.CreateAdd(cost, _additionalCost);
		
		m_checkCall->setArgOperand(0, cost); // Update block cost in gas check call
		m_checkCall = nullptr; // End cost-block
		m_blockCost = 0;
	}
	assert(m_blockCost == 0);
}

void GasMeter::checkMemory(llvm::Value* _additionalMemoryInWords, llvm::IRBuilder<>& _builder)
{
	// Memory uses other builder, but that can be changes later
	auto cost = _builder.CreateMul(_additionalMemoryInWords, Constant::get(static_cast<uint64_t>(c_memoryGas)), "memcost");
	_builder.CreateCall(m_gasCheckFunc, cost);
}

}