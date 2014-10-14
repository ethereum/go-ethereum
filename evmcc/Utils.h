
#pragma once

#include <llvm/IR/IRBuilder.h>

#include <libdevcore/Common.h>

namespace evmcc
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

dev::u256 llvm2eth(i256);
i256 eth2llvm(dev::u256);

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

}