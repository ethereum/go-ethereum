#include "Arith256.h"
#include "Runtime.h"
#include "Type.h"

#include <llvm/IR/Function.h>

namespace dev
{
namespace eth
{
namespace jit
{

Arith256::Arith256(llvm::IRBuilder<>& _builder) :
	CompilerHelper(_builder)
{
	using namespace llvm;

	m_result = m_builder.CreateAlloca(Type::Word, nullptr, "arith.result");
	m_arg1 = m_builder.CreateAlloca(Type::Word, nullptr, "arith.arg1");
	m_arg2 = m_builder.CreateAlloca(Type::Word, nullptr, "arith.arg2");
	m_arg3 = m_builder.CreateAlloca(Type::Word, nullptr, "arith.arg3");

	using Linkage = GlobalValue::LinkageTypes;

	llvm::Type* arg2Types[] = {Type::WordPtr, Type::WordPtr, Type::WordPtr};
	llvm::Type* arg3Types[] = {Type::WordPtr, Type::WordPtr, Type::WordPtr, Type::WordPtr};

	m_mul = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_mul", getModule());
	m_div = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_div", getModule());
	m_mod = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_mod", getModule());
	m_sdiv = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_sdiv", getModule());
	m_smod = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_smod", getModule());
	m_exp = Function::Create(FunctionType::get(Type::Void, arg2Types, false), Linkage::ExternalLinkage, "arith_exp", getModule());
	m_addmod = Function::Create(FunctionType::get(Type::Void, arg3Types, false), Linkage::ExternalLinkage, "arith_addmod", getModule());
	m_mulmod = Function::Create(FunctionType::get(Type::Void, arg3Types, false), Linkage::ExternalLinkage, "arith_mulmod", getModule());
}

Arith256::~Arith256()
{}

llvm::Value* Arith256::binaryOp(llvm::Function* _op, llvm::Value* _arg1, llvm::Value* _arg2)
{
	m_builder.CreateStore(_arg1, m_arg1);
	m_builder.CreateStore(_arg2, m_arg2);
	m_builder.CreateCall3(_op, m_arg1, m_arg2, m_result);
	return m_builder.CreateLoad(m_result);
}

llvm::Value* Arith256::ternaryOp(llvm::Function* _op, llvm::Value* _arg1, llvm::Value* _arg2, llvm::Value* _arg3)
{
	m_builder.CreateStore(_arg1, m_arg1);
	m_builder.CreateStore(_arg2, m_arg2);
	m_builder.CreateStore(_arg3, m_arg3);
	m_builder.CreateCall4(_op, m_arg1, m_arg2, m_arg3, m_result);
	return m_builder.CreateLoad(m_result);
}

llvm::Value* Arith256::mul(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_mul, _arg1, _arg2);
}

llvm::Value* Arith256::div(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_div, _arg1, _arg2);
}

llvm::Value* Arith256::mod(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_mod, _arg1, _arg2);
}

llvm::Value* Arith256::sdiv(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_sdiv, _arg1, _arg2);
}

llvm::Value* Arith256::smod(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_smod, _arg1, _arg2);
}

llvm::Value* Arith256::exp(llvm::Value* _arg1, llvm::Value* _arg2)
{
	return binaryOp(m_exp, _arg1, _arg2);
}

llvm::Value* Arith256::addmod(llvm::Value* _arg1, llvm::Value* _arg2, llvm::Value* _arg3)
{
	return ternaryOp(m_addmod, _arg1, _arg2, _arg3);
}

llvm::Value* Arith256::mulmod(llvm::Value* _arg1, llvm::Value* _arg2, llvm::Value* _arg3)
{
	return ternaryOp(m_mulmod, _arg1, _arg2, _arg3);
}

namespace
{
	using s256 = boost::multiprecision::int256_t;

	inline s256 u2s(u256 _u)
	{
		static const bigint c_end = (bigint)1 << 256;
		static const u256 c_send = (u256)1 << 255;
		if (_u < c_send)
			return (s256)_u;
		else
			return (s256)-(c_end - _u);
	}

	inline u256 s2u(s256 _u)
	{
		static const bigint c_end = (bigint)1 << 256;
		if (_u >= 0)
			return (u256)_u;
		else
			return (u256)(c_end + _u);
	}
}

}
}
}


extern "C"
{

	using namespace dev::eth::jit;

	EXPORT void arith_mul(i256* _arg1, i256* _arg2, i256* o_result)
	{
		auto arg1 = llvm2eth(*_arg1);
		auto arg2 = llvm2eth(*_arg2);
		*o_result = eth2llvm(arg1 * arg2);
	}

	EXPORT void arith_div(i256* _arg1, i256* _arg2, i256* o_result)
	{
		auto arg1 = llvm2eth(*_arg1);
		auto arg2 = llvm2eth(*_arg2);
		*o_result = eth2llvm(arg2 == 0 ? arg2 : arg1 / arg2);
	}

	EXPORT void arith_mod(i256* _arg1, i256* _arg2, i256* o_result)
	{
		auto arg1 = llvm2eth(*_arg1);
		auto arg2 = llvm2eth(*_arg2);
		*o_result = eth2llvm(arg2 == 0 ? arg2 : arg1 % arg2);
	}

	EXPORT void arith_sdiv(i256* _arg1, i256* _arg2, i256* o_result)
	{
		auto arg1 = llvm2eth(*_arg1);
		auto arg2 = llvm2eth(*_arg2);
		*o_result = eth2llvm(arg2 == 0 ? arg2 : s2u(u2s(arg1) / u2s(arg2)));
	}

	EXPORT void arith_smod(i256* _arg1, i256* _arg2, i256* o_result)
	{
		auto arg1 = llvm2eth(*_arg1);
		auto arg2 = llvm2eth(*_arg2);
		*o_result = eth2llvm(arg2 == 0 ? arg2 : s2u(u2s(arg1) % u2s(arg2)));
	}

	EXPORT void arith_exp(i256* _arg1, i256* _arg2, i256* o_result)
	{
		bigint left = llvm2eth(*_arg1);
		bigint right = llvm2eth(*_arg2);
		auto ret = static_cast<u256>(boost::multiprecision::powm(left, right, bigint(2) << 256));
		*o_result = eth2llvm(ret);
	}

	EXPORT void arith_mulmod(i256* _arg1, i256* _arg2, i256* _arg3, i256* o_result)
	{
		auto arg1 = llvm2eth(*_arg1);
		auto arg2 = llvm2eth(*_arg2);
		auto arg3 = llvm2eth(*_arg3);
		if (arg3 != 0)
			*o_result = eth2llvm(u256((bigint(arg1) * bigint(arg2)) % arg3));
		else
			*o_result = {};
	}

	EXPORT void arith_addmod(i256* _arg1, i256* _arg2, i256* _arg3, i256* o_result)
	{
		auto arg1 = llvm2eth(*_arg1);
		auto arg2 = llvm2eth(*_arg2);
		auto arg3 = llvm2eth(*_arg3);
		if (arg3 != 0)
			*o_result = eth2llvm(u256((bigint(arg1) + bigint(arg2)) % arg3));
		else
			*o_result = {};
	}

}


