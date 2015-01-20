#include "Arith256.h"
#include "Runtime.h"
#include "Type.h"
#include "Endianness.h"

#include <llvm/IR/Function.h>
#include <gmp.h>

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

	//return Endianness::toNative(m_builder, binaryOp(m_div, Endianness::toBE(m_builder, _arg1), Endianness::toBE(m_builder, _arg2)));
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
	using uint128 = __uint128_t;

//	uint128 add(uint128 a, uint128 b) { return a + b; }
//	uint128 mul(uint128 a, uint128 b) { return a * b; }
//
//	uint128 mulq(uint64_t x, uint64_t y)
//	{
//		return (uint128)x * (uint128)y;
//	}
//
//	uint128 addc(uint64_t x, uint64_t y)
//	{
//		return (uint128)x * (uint128)y;
//	}

	struct uint256
	{
		uint64_t lo;
		uint64_t mid;
		uint128 hi;
	};

//	uint256 add(uint256 x, uint256 y)
//	{
//		auto lo = (uint128) x.lo + y.lo;
//		auto mid = (uint128) x.mid + y.mid + (lo >> 64);
//		return {lo, mid, x.hi + y.hi + (mid >> 64)};
//	}

	uint256 mul(uint256 x, uint256 y)
	{
		auto t1 = (uint128) x.lo * y.lo;
		auto t2 = (uint128) x.lo * y.mid;
		auto t3 = x.lo * y.hi;
		auto t4 = (uint128) x.mid * y.lo;
		auto t5 = (uint128) x.mid * y.mid;
		auto t6 = x.mid * y.hi;
		auto t7 = x.hi * y.lo;
		auto t8 = x.hi * y.mid;

		auto lo = (uint64_t) t1;
		auto m1 = (t1 >> 64) + (uint64_t) t2;
		auto m2 = (uint64_t) m1;
		auto mid = (uint128) m2 + (uint64_t) t4;
		auto hi = (t2 >> 64) + t3 + (t4 >> 64) + t5 + (t6 << 64) + t7
			 + (t8 << 64) + (m1 >> 64) + (mid >> 64);

		return {lo, (uint64_t)mid, hi};
	}

	bool isZero(i256 const* _n)
	{
		return _n->a == 0 && _n->b == 0 && _n->c == 0 && _n->d == 0;
	}

	const auto nLimbs = sizeof(i256) / sizeof(mp_limb_t);

	// FIXME: Not thread-safe
	static mp_limb_t mod_limbs[] = {0, 0, 0, 0, 1};
	static_assert(sizeof(mod_limbs) / sizeof(mod_limbs[0]) == nLimbs + 1, "mp_limb_t size mismatch");
	static const mpz_t mod{nLimbs + 1, nLimbs + 1, &mod_limbs[0]};

	static mp_limb_t tmp_limbs[nLimbs + 2];
	static mpz_t tmp{nLimbs + 2, 0, &tmp_limbs[0]};

	int countLimbs(i256 const* _n)
	{
		static const auto limbsInWord = sizeof(_n->a) / sizeof(mp_limb_t);
		static_assert(limbsInWord == 1, "E?");

		int l = nLimbs;
		if (_n->d != 0) return l;
		l -= limbsInWord;
		if (_n->c != 0) return l;
		l -= limbsInWord;
		if (_n->b != 0) return l;
		l -= limbsInWord;
		if (_n->a != 0) return l;
		return 0;
	}

	void u2s(mpz_t _u)
	{
		if (static_cast<std::make_signed<mp_limb_t>::type>(_u->_mp_d[nLimbs - 1]) < 0)
		{
			mpz_sub(tmp, mod, _u);
			mpz_set(_u, tmp);
			_u->_mp_size = -_u->_mp_size;
		}
	}

	void s2u(mpz_t _s)
	{
		if (_s->_mp_size < 0)
		{
			mpz_add(tmp, mod, _s);
			mpz_set(_s, tmp);
		}
	}
}

}
}
}


extern "C"
{

	using namespace dev::eth::jit;

	EXPORT void arith_mul(uint256* _arg1, uint256* _arg2, uint256* o_result)
	{
		*o_result = mul(*_arg1, *_arg2);
	}

	EXPORT void arith_div(i256* _arg1, i256* _arg2, i256* o_result)
	{
		*o_result = {};
		if (isZero(_arg2))
			return;

		mpz_t x{nLimbs, countLimbs(_arg1), reinterpret_cast<mp_limb_t*>(_arg1)};
		mpz_t y{nLimbs, countLimbs(_arg2), reinterpret_cast<mp_limb_t*>(_arg2)};
		mpz_t z{nLimbs, 0, reinterpret_cast<mp_limb_t*>(o_result)};

		mpz_tdiv_q(z, x, y);

//		auto arg1 = llvm2eth(*_arg1);
//		auto arg2 = llvm2eth(*_arg2);
//		auto res = arg2 == 0 ? arg2 : arg1 / arg2;
//		std::cout << "DIV " << arg1 << "/" << arg2 << " = " << res << std::endl;
//		gmp_printf("GMP %Zd / %Zd = %Zd\n", x, y, z);
	}

	EXPORT void arith_mod(i256* _arg1, i256* _arg2, i256* o_result)
	{
		*o_result = {};
		if (isZero(_arg2))
			return;

		mpz_t x{nLimbs, countLimbs(_arg1), reinterpret_cast<mp_limb_t*>(_arg1)};
		mpz_t y{nLimbs, countLimbs(_arg2), reinterpret_cast<mp_limb_t*>(_arg2)};
		mpz_t z{nLimbs, 0, reinterpret_cast<mp_limb_t*>(o_result)};

		mpz_tdiv_r(z, x, y);
	}

	EXPORT void arith_sdiv(i256* _arg1, i256* _arg2, i256* o_result)
	{
		*o_result = {};
		if (isZero(_arg2))
			return;

		mpz_t x{nLimbs, countLimbs(_arg1), reinterpret_cast<mp_limb_t*>(_arg1)};
		mpz_t y{nLimbs, countLimbs(_arg2), reinterpret_cast<mp_limb_t*>(_arg2)};
		mpz_t z{nLimbs, 0, reinterpret_cast<mp_limb_t*>(o_result)};
		u2s(x);
		u2s(y);
		mpz_tdiv_q(z, x, y);
		s2u(z);
	}

	EXPORT void arith_smod(i256* _arg1, i256* _arg2, i256* o_result)
	{
		*o_result = {};
		if (isZero(_arg2))
			return;

		mpz_t x{nLimbs, countLimbs(_arg1), reinterpret_cast<mp_limb_t*>(_arg1)};
		mpz_t y{nLimbs, countLimbs(_arg2), reinterpret_cast<mp_limb_t*>(_arg2)};
		mpz_t z{nLimbs, 0, reinterpret_cast<mp_limb_t*>(o_result)};
		u2s(x);
		u2s(y);
		mpz_tdiv_r(z, x, y);
		s2u(z);
	}

	EXPORT void arith_exp(i256* _arg1, i256* _arg2, i256* o_result)
	{
		*o_result = {};

		static mp_limb_t mod_limbs[nLimbs + 1] = {};
		mod_limbs[nLimbs] = 1;
		static const mpz_t mod{nLimbs + 1, nLimbs + 1, &mod_limbs[0]};

		mpz_t x{nLimbs, countLimbs(_arg1), reinterpret_cast<mp_limb_t*>(_arg1)};
		mpz_t y{nLimbs, countLimbs(_arg2), reinterpret_cast<mp_limb_t*>(_arg2)};
		mpz_t z{nLimbs, 0, reinterpret_cast<mp_limb_t*>(o_result)};

		mpz_powm(z, x, y, mod);
	}

	EXPORT void arith_mulmod(i256* _arg1, i256* _arg2, i256* _arg3, i256* o_result)
	{
		*o_result = {};
		if (isZero(_arg3))
			return;

		mpz_t x{nLimbs, countLimbs(_arg1), reinterpret_cast<mp_limb_t*>(_arg1)};
		mpz_t y{nLimbs, countLimbs(_arg2), reinterpret_cast<mp_limb_t*>(_arg2)};
		mpz_t m{nLimbs, countLimbs(_arg3), reinterpret_cast<mp_limb_t*>(_arg3)};
		mpz_t z{nLimbs, 0, reinterpret_cast<mp_limb_t*>(o_result)};
		static mp_limb_t p_limbs[nLimbs * 2] = {};
		static mpz_t p{nLimbs * 2, 0, &p_limbs[0]};

		mpz_mul(p, x, y);
		mpz_tdiv_r(z, p, m);
	}

	EXPORT void arith_addmod(i256* _arg1, i256* _arg2, i256* _arg3, i256* o_result)
	{
		*o_result = {};
		if (isZero(_arg3))
			return;

		mpz_t x{nLimbs, countLimbs(_arg1), reinterpret_cast<mp_limb_t*>(_arg1)};
		mpz_t y{nLimbs, countLimbs(_arg2), reinterpret_cast<mp_limb_t*>(_arg2)};
		mpz_t m{nLimbs, countLimbs(_arg3), reinterpret_cast<mp_limb_t*>(_arg3)};
		mpz_t z{nLimbs, 0, reinterpret_cast<mp_limb_t*>(o_result)};
		static mp_limb_t s_limbs[nLimbs + 1] = {};
		static mpz_t s{nLimbs + 1, 0, &s_limbs[0]};

		mpz_add(s, x, y);
		mpz_tdiv_r(z, s, m);
	}

}


