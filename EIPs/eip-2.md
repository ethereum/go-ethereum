## <center>EIP-2: Homestead Hard-fork 분석</center>

<div style="text-align: right"> Seungmin Kim </div>
<div style="text-align: right"> 2024-04-04 </div>



### 개요
이 문서는 오라클 일반세션 오픈소스 팀 EIP 분석 프로젝트의 일환으로 EIP-2 에서 제안하는 내용과 그 분석에 대해서 서술합니다.


### 변경사항. 

#### 1. 트랜잭션을 통해 스마트계약을 생성하는 경우의 가스 비용을 21,000 에서 53,000으로 증가시킵니다. 

트랜잭션을 보내고 받는 사람 주소가 빈 문자열인 경우 차감되는 초기 가스는 현재의 경우인 21,000이 아니라 53,000에 tx 데이터의 가스 비용을 더한 금액입니다. CREATE opcode를 사용한 계약 생성은 영향을 받지 않습니다.

스마트 계약을 생성하는 방식은 두개가 존재합니다. 

1. 트랜잭션 To 를 빈 주소(0x0)로 전송하는 방법: 21,000 gas

2. CREATE opcode를 사용하는 방법: 32,000 gas

홈스테드 이전의 이더리움 버전의 경우 비용이 32,000인 CREATE opcode 보다 비용이 21,000인 거래를 통해 계약을 생성하는 데 초과 인센티브가 있습니다. 

또한 가스비용이 저렴하기 때문에, 자살 환불(suicide refunds)방법을 통해  11,664개의 가스만 사용하여 이더리움을 전송할 수 있었습니다.

자살 환불(suicide refunds)방법은 환불되는 가스의 주소를 전송할 주소로 지정하여 계약을 배포하는 방법입니다. 

<b>자살환불을 시도하는 python code</b>
```python
from ethereum import tester as t
> from ethereum import utils
> s = t.state()
> c = s.abi_contract('def init():\n suicide(0x47e25df8822538a8596b28c637896b4d143c351e)', endowment=10**15)
> s.block.get_receipts()[-1].gas_used
11664
>s.block.get_balance(utils.normalize_address(0x47e25df8822538a8596b28c637896b4d143c351e))
1000000000000000
```

<b>code</b>
```go
TxGas                 uint64 = 21000 // Per transaction not creating a contract. NOTE: Not payable on data of calls between transactions.
TxGasContractCreation uint64 = 53000 // Per transaction that creates a contract. NOTE: Not payable on data of calls between transactions.

...

CreateGas             uint64 = 32000 // Once per CREATE operation & contract-creation transaction.

...

SelfdestructRefundGas uint64 = 24000 // suicide 는 현재 selfdestruct 로 변경되었음.
```

```go
// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(data []byte, accessList types.AccessList, isContractCreation bool, isHomestead, isEIP2028, isEIP3860 bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	//eip2: 계약 생성 트랜잭션의 경우 가스비를 추가함.
	if isContractCreation && isHomestead {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
...
```

#### 2. S 값이 (secp256k1 ∙ n)/2 보다 큰 모든 거래 서명은 이제 유효하지 않은 것으로 간주됩니다. 

홈스테드 이전 버전에서는 공격자가 원래의 서명을 통해 새로운 유효한 서명을 생성할 수 있습니다. 이더리움의 ECDSA 서명은 (r,s) 의 쌍 값과 복구 식별자 'v' 를 가집니다. 그러나 이 서명 값을 알고 있다면,유효한 서명을 s를 사용하여 하나 더 생성할 수 있습니다.

(r, n - s) 와 v를 뒤집으면(27-> 28, 28->27), 새로운 유효한 서명이 생성됩니다.

이더리움은 이더 가치 전송이나 다른 거래에 대한 입력으로 주소를 사용하기 때문에 이는 심각한 보안 결함은 아니지만 공격자가 거래를 방해할 수 있으므로 불편함을 초래할 수 있습니다. 

```go
func VerifySignature(pubkey, hash, signature []byte) bool {
	if len(signature) != 64 {
		return false
	}
	var r, s btcec.ModNScalar
	if r.SetByteSlice(signature[:32]) {
		return false // overflow
	}
	if s.SetByteSlice(signature[32:]) {
		return false
	}
	sig := btc_ecdsa.NewSignature(&r, &s)
	key, err := btcec.ParsePubKey(pubkey)
	if err != nil {
		return false
	}
	// Reject malleable signatures. libsecp256k1 does this check but btcec doesn't.
	if s.IsOverHalfOrder() {
		return false
	}
	return sig.Verify(hash, key)
}
```



#### 3. 계약 생성 프로세스에서 계약 코드를 상태에 추가하기 위한 최종 가스 요금을 지불할 만큼 가스가 충분하지 않은 경우 빈 계약을 유지하는 대신 계약 생성이 실패되도록 합니다. 

최종 가스 수수료를 지불할 가스가 충분하지 않은 경우 계약 생성에 실패하면, 다음과 같은 이점이 있습니다.

(i) 현재의 "성공, 실패 또는 빈 계약"의 결과 계약 생성 프로세스의 결과에 보다 직관적인 "성공 또는 실패" 결과만을  생성합니다.

(ii) 계약 생성이 완전히 성공하지 않으면 계약 계정이 전혀 생성되지 않으므로 생성 실패를 더 쉽게 감지할 수 있습니다. 그리고

(iii) 거래가 실패하더라도, 가스가 환불된다는 보장이 있으므로 계약 생성을 더 안전하게 만듭니다.

<b>code</b>
```go
	// We first execute the transaction at the highest allowable gas limit, since if this fails we
	// can return error immediately.
	failed, result, err := execute(ctx, call, opts, hi)
	if err != nil {
		return 0, nil, err
	}
	if failed {
		if result != nil && !errors.Is(result.Err, vm.ErrOutOfGas) {
			return 0, result.Revert(), result.Err
		}
		return 0, nil, fmt.Errorf("gas required exceeds allowance (%d)", hi)
	}
```



#### 4. 현재 공식에서 난이도 조정 알고리즘을 변경합니다: 

$$
\text{block\_diff} = \text{parent\_diff} + \frac{\text{parent\_diff}}{2048} \times \begin{cases} 
1, & \text{if block\_timestamp} - \text{parent\_timestamp} < 13 \\
-1, & \text{otherwise}
\end{cases} + \text{int}\left(2^{\left(\frac{\text{block.number}}{100000} - 2\right)}\right)
$$

를 

$$
\text{block\_diff} = \text{parent\_diff} + \frac{\text{parent\_diff}}{2048} \times \max\left(1 - \frac{\text{block\_timestamp} - \text{parent\_timestamp}}{10}, -99\right) + \text{int}\left(2^{\left(\frac{\text{block.number}}{100000} - 2\right)}\right)
$$

로 변경합니다. 최소 난이도는 그대로 유지되며 최소 난이도 아래로는 변경되지 않습니다. 

기존의 난이도 알고리즘은 네트워크의 해시레이크가 급격하게 변하는 경우, 빠르게 이에 적응하는 것이 어렵습니다. 때문에 2015년 9월 이더리움 프로토콜에서 과도한 수의 채굴자가 $\text{parent\_timestamp} + 1$ 의 타임스탬프 값(1초의 블록 생성 시간을)을 가지는 블록들을 채굴하던 문제가 있었고, 이는 블록 시간 분포를 왜곡 시켰습니다.



따라서 새로운 난이도 알고리즘은 이러한 문제를 해결하도록 제안되었습니다. 제안된 새로운 공식은 대략적으로 평균값을 목표로 하고 있으며, 이 공식은  24초보다 긴 평균 블록 시간이 수학적으로 불가능하다는 것을 증명할 수 있습니다.

시간 차이가 아닌 $\frac{block\_timestamp - parent\_timestamp}{10}$을 주요 입력 변수로 사용하는 것은 알고리즘의 대략적인 특성을 직접적으로 유지하는 역할을 합니다. 

이는 채굴자들이 약간 높은 난이도를 가진 블록을 생성하여 가능한 모든 포크를 확실히 이길 수 있도록 타임스탬프 차이를 정확히 $1$로 설정하는 과도한 유인을 방지합니다.

$-99$의 상한은 두 블록이 시간상 매우 멀리 떨어져 있는 경우 난이도가 크게 떨어지지 않도록 하는 역할을 합니다.

#### Code 
<b>Homestead</b>

```go
func calcDifficultyHomestead(time, parentTime uint64, parentNumber, parentDiff *big.Int) *big.Int {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.mediawiki
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).SetUint64(parentTime)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp -parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big10)
	x.Sub(common.Big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)))
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}

	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parentDiff, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parentDiff, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(params.MinimumDifficulty) < 0 {
		x = params.MinimumDifficulty
	}

	// for the exponential factor
	periodCount := new(big.Int).Add(parentNumber, common.Big1)
	periodCount.Div(periodCount, ExpDiffPeriod)

	// the exponential factor, commonly refered to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(common.Big1) > 0 {
		y.Sub(periodCount, common.Big2)
		y.Exp(common.Big2, y, nil)
		x.Add(x, y)
	}

	return x
}
```

<b>Frontier</b>

```go
func calcDifficultyFrontier(time, parentTime uint64, parentNumber, parentDiff *big.Int) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Div(parentDiff, params.DifficultyBoundDivisor)
	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.SetUint64(parentTime)

	if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
		diff.Add(parentDiff, adjust)
	} else {
		diff.Sub(parentDiff, adjust)
	}
	if diff.Cmp(params.MinimumDifficulty) < 0 {
		diff = params.MinimumDifficulty
	}

	periodCount := new(big.Int).Add(parentNumber, common.Big1)
	periodCount.Div(periodCount, ExpDiffPeriod)
	if periodCount.Cmp(common.Big1) > 0 {
		// diff = diff + 2^(periodCount - 2)
		expDiff := periodCount.Sub(periodCount, common.Big2)
		expDiff.Exp(common.Big2, expDiff, nil)
		diff.Add(diff, expDiff)
		diff = common.BigMax(diff, params.MinimumDifficulty)
	}

	return diff
}
```

<b>Reference</b>

https://eips.ethereum.org/EIPS/eip-2

https://medium.com/coinmonks/learn-solidity-lesson-37-creating-and-destroying-contracts-6921ae32413a

https://github.com/ethereum/go-ethereum/compare/v1.3.3...v1.3.
