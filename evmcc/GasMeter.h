
#pragma once

#include <llvm/IR/IRBuilder.h>

#include <libevmface/Instruction.h>

namespace evmcc
{

class GasMeter
{
public:
	GasMeter(llvm::IRBuilder<>& _builder, llvm::Module* module);

	GasMeter(const GasMeter&) = delete;
	void operator=(GasMeter) = delete;

	void check(dev::eth::Instruction _inst);

private:
	llvm::IRBuilder<>& m_builder;
	llvm::GlobalVariable* m_gas;
	llvm::Function* m_gasCheckFunc;
};

}