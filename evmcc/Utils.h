
#pragma once

#include <llvm/IR/IRBuilder.h>

#include <libdevcore/Common.h>

namespace dev
{
namespace eth
{
namespace jit
{

/// Representation of 256-bit value binary compatible with LLVM i256
// TODO: Replace with h256
struct i256
{
	uint64_t a;
	uint64_t b;
	uint64_t c;
	uint64_t d;
};
static_assert(sizeof(i256) == 32, "Wrong i265 size");

u256 llvm2eth(i256);
i256 eth2llvm(u256);

struct InsertPointGuard
{
	InsertPointGuard(llvm::IRBuilder<>& _builder) :
		m_builder(_builder),
		m_insertBB(m_builder.GetInsertBlock()),
		m_insertPt(m_builder.GetInsertPoint())
	{}

	~InsertPointGuard()
	{
		m_builder.SetInsertPoint(m_insertBB, m_insertPt);
	}

private:
	llvm::IRBuilder<>& m_builder;
	llvm::BasicBlock* m_insertBB;
	llvm::BasicBlock::iterator m_insertPt;

	InsertPointGuard(const InsertPointGuard&) = delete;
	void operator=(InsertPointGuard) = delete;
};

#define ANY_PUSH	  PUSH1:  \
	case Instruction::PUSH2:  \
	case Instruction::PUSH3:  \
	case Instruction::PUSH4:  \
	case Instruction::PUSH5:  \
	case Instruction::PUSH6:  \
	case Instruction::PUSH7:  \
	case Instruction::PUSH8:  \
	case Instruction::PUSH9:  \
	case Instruction::PUSH10: \
	case Instruction::PUSH11: \
	case Instruction::PUSH12: \
	case Instruction::PUSH13: \
	case Instruction::PUSH14: \
	case Instruction::PUSH15: \
	case Instruction::PUSH16: \
	case Instruction::PUSH17: \
	case Instruction::PUSH18: \
	case Instruction::PUSH19: \
	case Instruction::PUSH20: \
	case Instruction::PUSH21: \
	case Instruction::PUSH22: \
	case Instruction::PUSH23: \
	case Instruction::PUSH24: \
	case Instruction::PUSH25: \
	case Instruction::PUSH26: \
	case Instruction::PUSH27: \
	case Instruction::PUSH28: \
	case Instruction::PUSH29: \
	case Instruction::PUSH30: \
	case Instruction::PUSH31: \
	case Instruction::PUSH32

#define ANY_DUP		  DUP1:	 \
	case Instruction::DUP2:	 \
	case Instruction::DUP3:	 \
	case Instruction::DUP4:	 \
	case Instruction::DUP5:	 \
	case Instruction::DUP6:	 \
	case Instruction::DUP7:	 \
	case Instruction::DUP8:	 \
	case Instruction::DUP9:	 \
	case Instruction::DUP10: \
	case Instruction::DUP11: \
	case Instruction::DUP12: \
	case Instruction::DUP13: \
	case Instruction::DUP14: \
	case Instruction::DUP15: \
	case Instruction::DUP16

#define ANY_SWAP	  SWAP1:  \
	case Instruction::SWAP2:  \
	case Instruction::SWAP3:  \
	case Instruction::SWAP4:  \
	case Instruction::SWAP5:  \
	case Instruction::SWAP6:  \
	case Instruction::SWAP7:  \
	case Instruction::SWAP8:  \
	case Instruction::SWAP9:  \
	case Instruction::SWAP10: \
	case Instruction::SWAP11: \
	case Instruction::SWAP12: \
	case Instruction::SWAP13: \
	case Instruction::SWAP14: \
	case Instruction::SWAP15: \
	case Instruction::SWAP16

}
}
}
