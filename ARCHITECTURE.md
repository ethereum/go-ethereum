# X Chain 技术架构文档

基于 Intel SGX 远程证明的以太坊兼容区块链

## 1. 概述

### 1.1 项目背景

X Chain 是一个基于 go-ethereum (Geth) 的新型区块链，使用 Intel SGX 远程证明替代传统的 PoS (Proof of Stake) 共识机制。通过 Gramine LibOS 运行时，所有节点在 SGX 可信执行环境 (TEE) 中运行，确保代码执行的正确性和数据的完整性。

### 1.2 核心特性

- **完全兼容以太坊主网**：兼容现有的智能合约和交易格式
- **SGX 远程证明共识**：不依赖 51% 多数同意，而是基于硬件可信执行环境的确定性共识
- **安全密钥管理**：通过预编译合约提供密钥创建、签名、验签、ECDH 等能力，私钥永不离开可信环境
- **硬件真随机数**：通过 SGX RDRAND 指令提供硬件级真随机数
- **数据一致性即共识**：任何节点修改数据都意味着硬分叉

### 1.3 链参数

| 参数 | 值 |
|------|-----|
| 链名称 | X |
| Chain ID | 762385986 (0x2d711642) |
| Chain ID 计算方式 | sha256("x") 前 4 字节 |

## 2. 系统架构

### 2.1 整体架构图

```
+------------------------------------------------------------------+
|                        X Chain 节点                               |
|  +------------------------------------------------------------+  |
|  |                    SGX Enclave (Gramine)                   |  |
|  |  +------------------------------------------------------+  |  |
|  |  |                   修改后的 Geth                       |  |  |
|  |  |  +----------------+  +----------------+              |  |  |
|  |  |  | SGX 共识引擎   |  | 预编译合约     |              |  |  |
|  |  |  | (PoA-SGX)      |  | (密钥管理)     |              |  |  |
|  |  |  +----------------+  +----------------+              |  |  |
|  |  |  +----------------+  +----------------+              |  |  |
|  |  |  | P2P 网络层     |  | EVM 执行层     |              |  |  |
|  |  |  | (RA-TLS)       |  |                |              |  |  |
|  |  |  +----------------+  +----------------+              |  |  |
|  |  +------------------------------------------------------+  |  |
|  |                           |                                |  |
|  |  +------------------------------------------------------+  |  |
|  |  |              Gramine 加密分区                         |  |  |
|  |  |  - 私钥存储                                          |  |  |
|  |  |  - 派生秘密 (ECDH 结果等)                            |  |  |
|  |  |  - 区块链数据                                        |  |  |
|  |  +------------------------------------------------------+  |  |
|  +------------------------------------------------------------+  |
+------------------------------------------------------------------+
                              |
                    RA-TLS 加密通道
                              |
+------------------------------------------------------------------+
|                      其他 X Chain 节点                            |
+------------------------------------------------------------------+
```

### 2.2 核心组件

#### 2.2.1 SGX 共识引擎 (PoA-SGX)

新的共识引擎实现 `consensus.Engine` 接口，基于 SGX 远程证明：

```go
// consensus/sgx/consensus.go
package sgx

type SGXConsensus struct {
    config     *params.SGXConfig
    attestor   *SGXAttestor      // SGX 远程证明器
    keyManager *KeyManager       // 密钥管理器
}

// 实现 consensus.Engine 接口
func (s *SGXConsensus) VerifyHeader(chain ChainHeaderReader, header *types.Header) error
func (s *SGXConsensus) Seal(chain ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error
func (s *SGXConsensus) Finalize(chain ChainHeaderReader, header *types.Header, state vm.StateDB, body *types.Body)
```

#### 2.2.2 预编译合约系统

新增预编译合约地址范围：`0x0200` - `0x02FF`

| 地址 | 功能 | 描述 |
|------|------|------|
| 0x0200 | SGX_KEY_CREATE | 创建密钥对 |
| 0x0201 | SGX_KEY_GET_PUBLIC | 获取公钥 |
| 0x0202 | SGX_SIGN | 签名 |
| 0x0203 | SGX_VERIFY | 验签 |
| 0x0204 | SGX_ECDH | ECDH 密钥交换 |
| 0x0205 | SGX_RANDOM | 硬件真随机数 |
| 0x0206 | SGX_ENCRYPT | 对称加密 |
| 0x0207 | SGX_DECRYPT | 对称解密 |
| 0x0208 | SGX_KEY_DERIVE | 密钥派生 |

#### 2.2.3 Gramine 运行时集成

节点通过 Gramine LibOS 在 SGX enclave 中运行：

```
gramine-sgx geth --datadir /app/wallet/chaindata --networkid 762385986
```

## 3. 共识机制详细设计

### 3.1 核心理念

X Chain 的共识机制基于以下核心原则：

1. **不依赖多数同意**：不使用 51% 权力维持共识
2. **确定性执行**：SGX 保证所有节点执行相同代码得到相同结果
3. **数据一致性即网络身份**：保持数据一致的节点属于同一网络
4. **修改即分叉**：任何节点修改数据都意味着硬分叉

### 3.2 节点身份验证

每个节点启动时必须通过 SGX 远程证明：

```
+-------------+                    +-------------+
|   新节点    |                    |  现有节点   |
+-------------+                    +-------------+
      |                                  |
      |  1. 请求加入网络                 |
      |--------------------------------->|
      |                                  |
      |  2. 发送 RA-TLS 证书请求         |
      |<---------------------------------|
      |                                  |
      |  3. 生成 SGX Quote               |
      |  (包含 MRENCLAVE, MRSIGNER)      |
      |                                  |
      |  4. 返回 RA-TLS 证书             |
      |--------------------------------->|
      |                                  |
      |  5. 验证 SGX Quote               |
      |  - 检查 MRENCLAVE (代码度量)     |
      |  - 检查 MRSIGNER (签名者)        |
      |  - 检查 TCB 状态                 |
      |                                  |
      |  6. 验证通过，允许加入           |
      |<---------------------------------|
      |                                  |
```

### 3.3 区块生产

区块生产采用轮询机制，所有通过 SGX 验证的节点都有权出块：

```go
// 区块头扩展字段
type SGXBlockHeader struct {
    // 标准以太坊区块头字段
    *types.Header
    
    // SGX 扩展字段 (存储在 Extra 中)
    SGXQuote      []byte  // 出块节点的 SGX Quote
    ProducerID    []byte  // 出块节点标识
    AttestationTS uint64  // 证明时间戳
}
```

### 3.4 区块验证流程

```go
func (s *SGXConsensus) VerifyHeader(chain ChainHeaderReader, header *types.Header) error {
    // 1. 验证基本区块头字段
    if err := s.verifyBasicHeader(header); err != nil {
        return err
    }
    
    // 2. 解析 Extra 字段中的 SGX 证明数据
    sgxData, err := s.parseSGXExtra(header.Extra)
    if err != nil {
        return err
    }
    
    // 3. 验证 SGX Quote
    if err := s.verifyQuote(sgxData.SGXQuote); err != nil {
        return err
    }
    
    // 4. 验证 MRENCLAVE 是否在白名单中
    if !s.isValidMREnclave(sgxData.MRENCLAVE) {
        return ErrInvalidMREnclave
    }
    
    // 5. 验证区块签名
    return s.verifyBlockSignature(header, sgxData)
}
```

## 4. 预编译合约详细设计

### 4.1 密钥管理架构

```
+------------------------------------------------------------------+
|                        密钥管理系统                               |
|  +------------------------------------------------------------+  |
|  |                     权限控制层                              |  |
|  |  - 密钥所有权验证 (msg.sender == keyOwner)                 |  |
|  |  - 操作权限检查                                            |  |
|  +------------------------------------------------------------+  |
|  |                     密钥操作层                              |  |
|  |  +------------+  +------------+  +------------+            |  |
|  |  | 签名/验签  |  | ECDH       |  | 加密/解密  |            |  |
|  |  +------------+  +------------+  +------------+            |  |
|  +------------------------------------------------------------+  |
|  |                     密钥存储层                              |  |
|  |  +------------------------------------------------------+  |  |
|  |  |              Gramine 加密分区                         |  |  |
|  |  |  /app/wallet/keys/{keyId}/                           |  |  |
|  |  |    - private.key (私钥，永不离开 enclave)            |  |  |
|  |  |    - public.key (公钥)                               |  |  |
|  |  |    - metadata.json (所有者、曲线类型等)              |  |  |
|  |  +------------------------------------------------------+  |  |
|  +------------------------------------------------------------+  |
+------------------------------------------------------------------+
```

### 4.2 支持的椭圆曲线

| 曲线名称 | 标识符 | 用途 |
|----------|--------|------|
| secp256k1 | 0x01 | 以太坊兼容签名 |
| secp256r1 (P-256) | 0x02 | TLS、通用签名 |
| secp384r1 (P-384) | 0x03 | 高安全性签名 |
| ed25519 | 0x04 | 高性能签名 |
| x25519 | 0x05 | ECDH 密钥交换 |

### 4.3 预编译合约接口定义

#### 4.3.1 SGX_KEY_CREATE (0x0200)

创建新的密钥对，私钥存储在加密分区。

**输入格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0      | 曲线类型 (1=secp256k1, 2=P-256, 3=P-384, 4=ed25519, 5=x25519) |
+--------+--------+
```

**输出格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | keyId (密钥标识符，sha256(owner || nonce)) |
+--------+--------+
```

**Gas 消耗：** 50000

**实现：**
```go
// core/vm/contracts_sgx.go
type sgxKeyCreate struct{}

func (c *sgxKeyCreate) RequiredGas(input []byte) uint64 {
    return 50000
}

func (c *sgxKeyCreate) Run(input []byte, caller common.Address, evm *EVM) ([]byte, error) {
    if len(input) < 1 {
        return nil, ErrInvalidInput
    }
    
    curveType := input[0]
    
    // 生成密钥对
    keyPair, err := generateKeyPair(curveType)
    if err != nil {
        return nil, err
    }
    
    // 计算 keyId
    nonce := evm.StateDB.GetNonce(caller)
    keyId := crypto.Keccak256Hash(caller.Bytes(), common.BigToHash(big.NewInt(int64(nonce))).Bytes())
    
    // 存储密钥到加密分区
    if err := storeKeyToEncryptedPartition(keyId, keyPair, caller); err != nil {
        return nil, err
    }
    
    // 记录密钥所有权到状态
    evm.StateDB.SetKeyOwner(keyId, caller)
    
    return keyId.Bytes(), nil
}
```

#### 4.3.2 SGX_KEY_GET_PUBLIC (0x0201)

获取指定密钥的公钥。

**输入格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | keyId  |
+--------+--------+
```

**输出格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0      | 曲线类型 |
| 1-N    | 公钥数据 (压缩或非压缩格式) |
+--------+--------+
```

**Gas 消耗：** 3000

#### 4.3.3 SGX_SIGN (0x0202)

使用私钥签名消息。

**输入格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | keyId  |
| 32-63  | 消息哈希 (32 字节) |
+--------+--------+
```

**输出格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-N    | 签名数据 (格式取决于曲线类型) |
+--------+--------+
```

**Gas 消耗：** 10000

**权限检查：**
```go
func (c *sgxSign) Run(input []byte, caller common.Address, evm *EVM) ([]byte, error) {
    keyId := common.BytesToHash(input[0:32])
    
    // 权限检查：只有密钥所有者可以签名
    owner := evm.StateDB.GetKeyOwner(keyId)
    if owner != caller {
        return nil, ErrNotKeyOwner
    }
    
    // 从加密分区加载私钥
    privateKey, err := loadPrivateKeyFromEncryptedPartition(keyId)
    if err != nil {
        return nil, err
    }
    
    // 签名
    messageHash := input[32:64]
    signature, err := sign(privateKey, messageHash)
    if err != nil {
        return nil, err
    }
    
    return signature, nil
}
```

#### 4.3.4 SGX_VERIFY (0x0203)

验证签名。

**输入格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0      | 曲线类型 |
| 1-N    | 公钥数据 |
| N+1-N+32 | 消息哈希 |
| N+33-M | 签名数据 |
+--------+--------+
```

**输出格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | 验证结果 (1=成功, 0=失败) |
+--------+--------+
```

**Gas 消耗：** 5000

#### 4.3.5 SGX_ECDH (0x0204)

执行 ECDH 密钥交换，派生的共享秘密存储在加密分区。

**输入格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | 本方私钥 keyId |
| 32     | 对方公钥曲线类型 |
| 33-N   | 对方公钥数据 |
+--------+--------+
```

**输出格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | 派生秘密的 keyId (可用于后续加密操作) |
+--------+--------+
```

**Gas 消耗：** 20000

**实现：**
```go
func (c *sgxECDH) Run(input []byte, caller common.Address, evm *EVM) ([]byte, error) {
    privateKeyId := common.BytesToHash(input[0:32])
    
    // 权限检查
    owner := evm.StateDB.GetKeyOwner(privateKeyId)
    if owner != caller {
        return nil, ErrNotKeyOwner
    }
    
    // 加载私钥
    privateKey, err := loadPrivateKeyFromEncryptedPartition(privateKeyId)
    if err != nil {
        return nil, err
    }
    
    // 解析对方公钥
    peerPublicKey, err := parsePublicKey(input[32:])
    if err != nil {
        return nil, err
    }
    
    // 执行 ECDH
    sharedSecret, err := ecdh(privateKey, peerPublicKey)
    if err != nil {
        return nil, err
    }
    
    // 派生秘密也遵循密钥管理逻辑，存储到加密分区
    derivedKeyId := crypto.Keccak256Hash(privateKeyId.Bytes(), peerPublicKey)
    if err := storeDerivedSecretToEncryptedPartition(derivedKeyId, sharedSecret, caller); err != nil {
        return nil, err
    }
    
    // 记录派生秘密所有权
    evm.StateDB.SetKeyOwner(derivedKeyId, caller)
    
    return derivedKeyId.Bytes(), nil
}
```

#### 4.3.6 SGX_RANDOM (0x0205)

获取硬件真随机数。

**输入格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | 请求的随机数长度 (最大 32 字节) |
+--------+--------+
```

**输出格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-N    | 随机数据 |
+--------+--------+
```

**Gas 消耗：** 1000 + 100 * 字节数

**实现：**
```go
func (c *sgxRandom) Run(input []byte) ([]byte, error) {
    length := new(big.Int).SetBytes(input).Uint64()
    if length > 32 {
        length = 32
    }
    
    // 使用 SGX RDRAND 指令获取硬件随机数
    randomBytes := make([]byte, length)
    if err := sgxRdrand(randomBytes); err != nil {
        return nil, err
    }
    
    return common.LeftPadBytes(randomBytes, 32), nil
}
```

#### 4.3.7 SGX_ENCRYPT (0x0206)

使用对称密钥加密数据。

**输入格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | keyId (对称密钥或 ECDH 派生密钥) |
| 32-N   | 明文数据 |
+--------+--------+
```

**输出格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-11   | Nonce (12 字节) |
| 12-N   | 密文 + Tag |
+--------+--------+
```

**Gas 消耗：** 5000 + 10 * 数据长度

#### 4.3.8 SGX_DECRYPT (0x0207)

使用对称密钥解密数据。

**输入格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | keyId |
| 32-43  | Nonce |
| 44-N   | 密文 + Tag |
+--------+--------+
```

**输出格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-N    | 明文数据 |
+--------+--------+
```

**Gas 消耗：** 5000 + 10 * 数据长度

#### 4.3.9 SGX_KEY_DERIVE (0x0208)

从现有密钥派生新密钥。

**输入格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | 源 keyId |
| 32-63  | 派生路径/盐值 |
+--------+--------+
```

**输出格式：**
```
+--------+--------+
| 字节   | 描述   |
+--------+--------+
| 0-31   | 新 keyId |
+--------+--------+
```

**Gas 消耗：** 10000

### 4.4 权限管理机制

#### 4.4.1 密钥所有权

每个密钥都有唯一的所有者（创建者的地址）：

```go
// 状态存储结构
type KeyMetadata struct {
    Owner     common.Address  // 密钥所有者
    CurveType uint8          // 曲线类型
    CreatedAt uint64         // 创建时间（区块号）
    KeyType   uint8          // 密钥类型 (0=非对称, 1=对称, 2=派生)
    ParentKey common.Hash    // 父密钥 (用于派生密钥)
}
```

#### 4.4.2 操作权限

| 操作 | 权限要求 |
|------|----------|
| 获取公钥 | 任何人 |
| 签名 | 仅所有者 |
| ECDH | 仅所有者 |
| 加密 | 仅所有者 |
| 解密 | 仅所有者 |
| 派生密钥 | 仅所有者 |

#### 4.4.3 派生秘密管理

ECDH 等操作产生的派生秘密也遵循相同的权限管理逻辑：

```go
// 派生秘密继承原始密钥的所有权
func (c *sgxECDH) Run(input []byte, caller common.Address, evm *EVM) ([]byte, error) {
    // ... ECDH 计算 ...
    
    // 派生秘密的所有者与原始私钥所有者相同
    evm.StateDB.SetKeyOwner(derivedKeyId, caller)
    evm.StateDB.SetKeyMetadata(derivedKeyId, KeyMetadata{
        Owner:     caller,
        KeyType:   2, // 派生密钥
        ParentKey: privateKeyId,
    })
    
    return derivedKeyId.Bytes(), nil
}
```

## 5. 数据存储与同步

### 5.1 存储架构

```
/app/wallet/                          # Gramine 加密分区根目录
├── chaindata/                        # 区块链数据
│   ├── ancient/                      # 历史数据
│   └── leveldb/                      # 当前状态
├── keys/                             # 密钥存储
│   └── {keyId}/
│       ├── private.key               # 私钥 (永不离开 enclave)
│       ├── public.key                # 公钥
│       └── metadata.json             # 元数据
├── derived/                          # 派生秘密
│   └── {derivedKeyId}/
│       └── secret.key
└── node/                             # 节点配置
    ├── nodekey                       # 节点私钥
    └── attestation/                  # 证明数据缓存
```

### 5.2 数据同步机制

#### 5.2.1 节点发现

与以太坊保持一致，使用 discv5 协议进行节点发现：

```go
// 节点发现时附加 SGX 证明信息
type SGXNodeRecord struct {
    *enode.Node
    MRENCLAVE []byte  // 代码度量值
    MRSIGNER  []byte  // 签名者度量值
    QuoteHash []byte  // 最新 Quote 哈希
}
```

#### 5.2.2 数据同步流程

```
+-------------+                    +-------------+
|   节点 A    |                    |   节点 B    |
+-------------+                    +-------------+
      |                                  |
      |  1. RA-TLS 握手                  |
      |<-------------------------------->|
      |  (双向 SGX 远程证明)             |
      |                                  |
      |  2. 交换区块头                   |
      |<-------------------------------->|
      |                                  |
      |  3. 验证数据一致性               |
      |  (比较状态根)                    |
      |                                  |
      |  4. 同步缺失区块                 |
      |<-------------------------------->|
      |                                  |
      |  5. 同步加密分区数据             |
      |  (密钥元数据，不含私钥)          |
      |<-------------------------------->|
      |                                  |
```

#### 5.2.3 加密分区数据同步

加密分区中的数据需要在节点间保持一致：

**同步的数据：**
- 密钥元数据（所有者、曲线类型、创建时间）
- 公钥数据
- 密钥所有权记录

**不同步的数据：**
- 私钥（每个节点独立生成，但通过 SGX sealing 保证一致性）

```go
// 密钥同步协议
type KeySyncMessage struct {
    KeyId     common.Hash
    Metadata  KeyMetadata
    PublicKey []byte
    // 私钥通过 SGX sealing 机制在各节点独立派生
    // 使用相同的 MRENCLAVE 和 sealing key 保证一致性
}
```

### 5.3 数据一致性验证

```go
// 验证两个节点是否属于同一网络
func (s *SGXConsensus) VerifyNetworkConsistency(peer *Peer) error {
    // 1. 比较创世区块哈希
    if peer.GenesisHash != s.genesisHash {
        return ErrDifferentGenesis
    }
    
    // 2. 比较最新区块状态根
    localHead := s.chain.CurrentHeader()
    peerHead := peer.Head()
    
    if localHead.Number.Cmp(peerHead.Number) == 0 {
        if localHead.Root != peerHead.Root {
            return ErrHardFork // 数据不一致，视为硬分叉
        }
    }
    
    // 3. 验证 MRENCLAVE 一致
    if !bytes.Equal(peer.MRENCLAVE, s.localMREnclave) {
        return ErrDifferentCode
    }
    
    return nil
}
```

## 6. P2P 网络层

### 6.1 节点连接准入控制

节点是否接受其他节点的连接和数据同步，取决于命令行配置的参数。只有满足以下条件的节点才能建立连接：

1. **度量值匹配**：对方节点的 MRENCLAVE/MRSIGNER 必须在允许列表中
2. **Chain ID 匹配**：对方节点的 Chain ID 必须与本节点一致

#### 6.1.0 设计目的：支持硬分叉升级

准入控制的核心目的是**支持硬分叉升级**，类似于以太坊的 EIP 实现机制。当需要发布新版代码实现新特性时，通过更新 MRENCLAVE 白名单来控制网络升级。

**硬分叉升级场景：**

1. **新特性发布**：类似 EIP-1559、EIP-4844 等协议升级，需要所有节点运行新版代码
2. **安全修复**：发现安全漏洞后，强制所有节点升级到修复版本
3. **性能优化**：优化后的代码产生不同的 MRENCLAVE，需要协调升级

**升级流程：**

```
时间线
  |
  v
+------------------+
| 阶段 1: 准备     |  发布新版代码，公布新 MRENCLAVE
+------------------+
  |
  v
+------------------+
| 阶段 2: 过渡     |  节点配置同时允许新旧 MRENCLAVE
|                  |  mrenclave = ["旧版本", "新版本"]
+------------------+
  |
  v
+------------------+
| 阶段 3: 升级     |  节点逐步升级到新版本
|                  |  新旧节点可以互相连接和同步
+------------------+
  |
  v
+------------------+
| 阶段 4: 完成     |  移除旧版 MRENCLAVE
|                  |  mrenclave = ["新版本"]
|                  |  未升级节点被隔离（硬分叉）
+------------------+
```

**版本兼容性管理示例：**

```toml
# config.toml - 过渡期配置
[sgx]
# 同时允许 v1.0.0 和 v1.1.0 版本
mrenclave = [
    "abc123...",  # v1.0.0 - 当前稳定版
    "def456...",  # v1.1.0 - 新版本（包含 XIP-001 特性）
]

# 升级完成后的配置
[sgx]
mrenclave = [
    "def456...",  # v1.1.0 - 仅允许新版本
]
# 运行 v1.0.0 的节点将无法连接，形成硬分叉
```

**与以太坊 EIP 的对比：**

| 特性 | 以太坊 EIP | X Chain XIP |
|------|-----------|-------------|
| 升级触发 | 区块高度 | MRENCLAVE 白名单 |
| 强制升级 | 需要社区共识 | 通过准入控制强制 |
| 回滚可能 | 困难 | 恢复旧 MRENCLAVE 即可 |
| 验证方式 | 区块验证规则 | SGX 远程证明 |

#### 6.1.1 命令行参数

```bash
# 启动 X Chain 节点
geth \
    --networkid 762385986 \
    --sgx.mrenclave "abc123...,def456..." \
    --sgx.mrsigner "789abc..." \
    --sgx.verify-mode "mrenclave" \
    --datadir /app/wallet/chaindata
```

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `--networkid` | Chain ID，必须匹配才能连接 | 762385986 |
| `--sgx.mrenclave` | 允许的 MRENCLAVE 列表（逗号分隔） | 本节点 MRENCLAVE |
| `--sgx.mrsigner` | 允许的 MRSIGNER 列表（逗号分隔） | 本节点 MRSIGNER |
| `--sgx.verify-mode` | 验证模式：`mrenclave`（严格）或 `mrsigner`（宽松） | `mrenclave` |
| `--sgx.tcb-allow-outdated` | 是否允许 TCB 过期的节点连接 | `false` |

#### 6.1.2 配置文件方式

```toml
# config.toml
[sgx]
# 允许的 MRENCLAVE 列表
mrenclave = [
    "abc123def456789...",  # v1.0.0 版本
    "def456789abc123...",  # v1.0.1 版本
]

# 允许的 MRSIGNER 列表
mrsigner = [
    "789abc123def456...",  # 官方签名者
]

# 验证模式
verify_mode = "mrenclave"  # 或 "mrsigner"

# TCB 策略
tcb_allow_outdated = false
```

#### 6.1.3 连接准入流程

```
+-------------+                    +-------------+
|   节点 A    |                    |   节点 B    |
+-------------+                    +-------------+
      |                                  |
      |  1. TCP 连接                     |
      |--------------------------------->|
      |                                  |
      |  2. RA-TLS 握手开始              |
      |<-------------------------------->|
      |                                  |
      |  3. 交换 SGX Quote               |
      |  (包含 MRENCLAVE, MRSIGNER)      |
      |<-------------------------------->|
      |                                  |
      |  4. 验证 Quote                   |
      |  - 检查 MRENCLAVE 是否在白名单   |
      |  - 检查 MRSIGNER 是否在白名单    |
      |  - 检查 TCB 状态                 |
      |                                  |
      |  5. 交换 Chain ID                |
      |<-------------------------------->|
      |                                  |
      |  6. 验证 Chain ID 匹配           |
      |  if (peerChainId != localChainId)|
      |      断开连接                    |
      |                                  |
      |  7. 连接建立成功                 |
      |<-------------------------------->|
      |                                  |
```

#### 6.1.4 准入控制实现

```go
// p2p/ratls/admission.go
package ratls

// AdmissionConfig 定义节点准入配置
type AdmissionConfig struct {
    ChainID          uint64    // Chain ID，必须匹配
    AllowedMREnclave [][]byte  // 允许的 MRENCLAVE 列表
    AllowedMRSigner  [][]byte  // 允许的 MRSIGNER 列表
    VerifyMode       string    // "mrenclave" 或 "mrsigner"
    AllowOutdatedTCB bool      // 是否允许 TCB 过期
}

// AdmissionController 控制节点连接准入
type AdmissionController struct {
    config *AdmissionConfig
}

func NewAdmissionController(config *AdmissionConfig) *AdmissionController {
    return &AdmissionController{config: config}
}

// VerifyPeer 验证对方节点是否允许连接
func (ac *AdmissionController) VerifyPeer(peerQuote []byte, peerChainID uint64) error {
    // 1. 验证 Chain ID
    if peerChainID != ac.config.ChainID {
        return fmt.Errorf("chain ID mismatch: expected %d, got %d", 
            ac.config.ChainID, peerChainID)
    }
    
    // 2. 解析 Quote
    mrenclave, mrsigner, tcbStatus, err := parseQuote(peerQuote)
    if err != nil {
        return fmt.Errorf("failed to parse quote: %w", err)
    }
    
    // 3. 验证 TCB 状态
    if !ac.config.AllowOutdatedTCB && tcbStatus != TCB_UP_TO_DATE {
        return fmt.Errorf("TCB status not up to date: %d", tcbStatus)
    }
    
    // 4. 根据验证模式检查度量值
    switch ac.config.VerifyMode {
    case "mrenclave":
        if !ac.isAllowedMREnclave(mrenclave) {
            return fmt.Errorf("MRENCLAVE not in allowed list: %x", mrenclave)
        }
    case "mrsigner":
        if !ac.isAllowedMRSigner(mrsigner) {
            return fmt.Errorf("MRSIGNER not in allowed list: %x", mrsigner)
        }
    default:
        // 默认使用 mrenclave 模式
        if !ac.isAllowedMREnclave(mrenclave) {
            return fmt.Errorf("MRENCLAVE not in allowed list: %x", mrenclave)
        }
    }
    
    return nil
}

func (ac *AdmissionController) isAllowedMREnclave(mrenclave []byte) bool {
    for _, allowed := range ac.config.AllowedMREnclave {
        if bytes.Equal(mrenclave, allowed) {
            return true
        }
    }
    return false
}

func (ac *AdmissionController) isAllowedMRSigner(mrsigner []byte) bool {
    for _, allowed := range ac.config.AllowedMRSigner {
        if bytes.Equal(mrsigner, allowed) {
            return true
        }
    }
    return false
}
```

### 6.2 RA-TLS 传输层

所有节点间通信使用 RA-TLS 加密通道：

```go
// p2p/ratls/transport.go
type RATLSTransport struct {
    localKey    *ecdsa.PrivateKey
    attestor    *SGXAttestor
    verifier    *SGXVerifier
    admission   *AdmissionController  // 准入控制器
}

func (t *RATLSTransport) Handshake(conn net.Conn) (*RATLSConn, error) {
    // 1. 生成 RA-TLS 证书
    cert, err := t.attestor.GenerateCertificate()
    if err != nil {
        return nil, err
    }
    
    // 2. TLS 握手
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        VerifyPeerCertificate: t.verifyPeerCertificate,
    }
    
    tlsConn := tls.Server(conn, tlsConfig)
    if err := tlsConn.Handshake(); err != nil {
        return nil, err
    }
    
    return &RATLSConn{Conn: tlsConn}, nil
}

func (t *RATLSTransport) verifyPeerCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {
    // 1. 解析证书中的 SGX Quote
    quote, err := extractSGXQuote(rawCerts[0])
    if err != nil {
        return err
    }
    
    // 2. 验证 Quote
    if err := t.verifier.VerifyQuote(quote); err != nil {
        return err
    }
    
    // 3. 检查 MRENCLAVE 是否在白名单中
    mrenclave := extractMREnclave(quote)
    if !t.isAllowedMREnclave(mrenclave) {
        return ErrInvalidMREnclave
    }
    
    return nil
}
```

### 6.2 消息协议扩展

```go
// eth/protocols/sgx/protocol.go
const (
    SGXProtocolName    = "sgx"
    SGXProtocolVersion = 1
)

// 新增消息类型
const (
    SGXStatusMsg          = 0x00  // SGX 状态信息
    SGXAttestationMsg     = 0x01  // 证明请求/响应
    SGXKeySyncMsg         = 0x02  // 密钥同步
    SGXConsistencyCheckMsg = 0x03 // 一致性检查
)

type SGXStatusPacket struct {
    MRENCLAVE     []byte
    MRSIGNER      []byte
    TCBStatus     uint8
    AttestationTS uint64
}
```

## 7. Gramine 集成

### 7.1 Manifest 配置

```toml
# geth.manifest.template

[libos]
entrypoint = "/usr/local/bin/geth"

[loader]
entrypoint = "file:{{ gramine.libos }}"
log_level = "warning"

[loader.env]
LD_LIBRARY_PATH = "/lib:/usr/lib:/usr/local/lib"
HOME = "/app"

[sys]
insecure__allow_eventfd = true
stack.size = "2M"
brk.max_size = "256M"

[sgx]
debug = false
enclave_size = "8G"
max_threads = 64
remote_attestation = "dcap"
trusted_files = [
    "file:/usr/local/bin/geth",
    "file:{{ gramine.libos }}",
    # ... 其他信任文件
]

# 加密分区配置
[[fs.mounts]]
type = "encrypted"
path = "/app/wallet"
uri = "file:/data/wallet"
key_name = "_sgx_mrenclave"

# 允许写入的文件
[sgx.allowed_files]
"/app/logs" = true
```

### 7.2 启动脚本

```bash
#!/bin/bash
# start-x-chain.sh

# 设置环境变量
export SGX_AESM_ADDR=1
export GRAMINE_LIBOS_PATH=/usr/lib/x86_64-linux-gnu/gramine/libsysdb.so

# 启动 Geth
exec gramine-sgx geth \
    --datadir /app/wallet/chaindata \
    --networkid 762385986 \
    --syncmode full \
    --gcmode archive \
    --http \
    --http.addr 0.0.0.0 \
    --http.port 8545 \
    --http.api eth,net,web3,sgx \
    --ws \
    --ws.addr 0.0.0.0 \
    --ws.port 8546 \
    --ws.api eth,net,web3,sgx
```

## 8. 模块拆解与实现指南

### 8.1 模块依赖关系

```
+------------------+
|  应用层 (RPC)    |
+------------------+
         |
+------------------+
|  预编译合约层    |
+------------------+
         |
+------------------+     +------------------+
|  共识引擎层      |<--->|  P2P 网络层      |
+------------------+     +------------------+
         |                        |
+------------------+     +------------------+
|  SGX 证明层      |     |  RA-TLS 层       |
+------------------+     +------------------+
         |                        |
+------------------+     +------------------+
|  Gramine 运行时  |<--->|  加密存储层      |
+------------------+     +------------------+
```

### 8.2 实现优先级

#### 第一阶段：基础设施

1. **Gramine 集成** (2 周)
   - 编写 Geth 的 Gramine manifest
   - 配置加密分区
   - 测试基本运行

2. **SGX 证明模块** (2 周)
   - 实现 SGX Quote 生成
   - 实现 Quote 验证
   - 集成 DCAP 库

#### 第二阶段：共识机制

3. **SGX 共识引擎** (3 周)
   - 实现 `consensus.Engine` 接口
   - 区块头扩展字段
   - 区块验证逻辑

4. **P2P RA-TLS 集成** (2 周)
   - 替换 RLPx 为 RA-TLS
   - 节点身份验证
   - 消息协议扩展

#### 第三阶段：预编译合约

5. **密钥管理预编译合约** (3 周)
   - SGX_KEY_CREATE
   - SGX_KEY_GET_PUBLIC
   - SGX_SIGN / SGX_VERIFY

6. **高级密码学预编译合约** (2 周)
   - SGX_ECDH
   - SGX_ENCRYPT / SGX_DECRYPT
   - SGX_KEY_DERIVE

7. **硬件随机数预编译合约** (1 周)
   - SGX_RANDOM

#### 第四阶段：数据同步

8. **加密分区数据同步** (2 周)
   - 密钥元数据同步协议
   - 一致性验证

9. **测试与优化** (2 周)
   - 单元测试
   - 集成测试
   - 性能优化

### 8.3 关键文件修改清单

```
go-ethereum/
├── consensus/
│   └── sgx/                          # 新增：SGX 共识引擎
│       ├── consensus.go              # 共识引擎实现
│       ├── attestor.go               # SGX 证明器
│       └── verifier.go               # Quote 验证器
├── core/
│   └── vm/
│       ├── contracts.go              # 修改：添加预编译合约注册
│       └── contracts_sgx.go          # 新增：SGX 预编译合约实现
├── p2p/
│   └── ratls/                        # 新增：RA-TLS 传输层
│       ├── transport.go
│       ├── handshake.go
│       └── certificate.go
├── eth/
│   └── protocols/
│       └── sgx/                      # 新增：SGX 协议
│           ├── protocol.go
│           ├── handler.go
│           └── peer.go
├── internal/
│   └── sgx/                          # 新增：SGX 内部工具
│       ├── keystore.go               # 加密分区密钥存储
│       ├── sealing.go                # SGX sealing
│       └── rdrand.go                 # 硬件随机数
└── params/
    └── config.go                     # 修改：添加 SGX 配置
```

### 8.4 接口定义

#### 8.4.1 SGX 证明器接口

```go
// internal/sgx/attestor.go
type Attestor interface {
    // 生成 SGX Quote
    GenerateQuote(reportData []byte) ([]byte, error)
    
    // 生成 RA-TLS 证书
    GenerateCertificate() (*tls.Certificate, error)
    
    // 获取本地 MRENCLAVE
    GetMREnclave() []byte
    
    // 获取本地 MRSIGNER
    GetMRSigner() []byte
}
```

#### 8.4.2 Quote 验证器接口

```go
// internal/sgx/verifier.go
type Verifier interface {
    // 验证 SGX Quote
    VerifyQuote(quote []byte) error
    
    // 验证 RA-TLS 证书
    VerifyCertificate(cert *x509.Certificate) error
    
    // 检查 MRENCLAVE 是否在白名单
    IsAllowedMREnclave(mrenclave []byte) bool
    
    // 添加 MRENCLAVE 到白名单
    AddAllowedMREnclave(mrenclave []byte)
}
```

#### 8.4.3 密钥存储接口

```go
// internal/sgx/keystore.go
type KeyStore interface {
    // 创建密钥对
    CreateKey(curveType uint8, owner common.Address) (common.Hash, error)
    
    // 获取公钥
    GetPublicKey(keyId common.Hash) ([]byte, error)
    
    // 签名
    Sign(keyId common.Hash, message []byte) ([]byte, error)
    
    // ECDH
    ECDH(keyId common.Hash, peerPublicKey []byte) (common.Hash, error)
    
    // 获取密钥所有者
    GetOwner(keyId common.Hash) (common.Address, error)
    
    // 验证所有权
    VerifyOwnership(keyId common.Hash, caller common.Address) bool
}
```

## 9. 安全考虑

### 9.1 威胁模型

| 威胁 | 缓解措施 |
|------|----------|
| 恶意节点运行篡改代码 | MRENCLAVE 验证确保代码完整性 |
| 私钥泄露 | 私钥存储在 SGX 加密分区，永不离开 enclave |
| 中间人攻击 | RA-TLS 双向认证 |
| 重放攻击 | Quote 包含时间戳和 nonce |
| 侧信道攻击 | 使用 SGX 最新安全补丁，避免敏感数据依赖的分支 |

### 9.2 密钥安全

1. **私钥隔离**：私钥永不离开 SGX enclave
2. **Sealing 保护**：使用 MRENCLAVE-based sealing 保护持久化密钥
3. **权限控制**：只有密钥所有者可以使用私钥
4. **派生秘密保护**：ECDH 等派生秘密同样存储在加密分区

### 9.3 网络安全

1. **双向认证**：所有节点通信都需要双向 SGX 远程证明
2. **MRENCLAVE 白名单**：只允许运行相同代码的节点加入网络
3. **TCB 检查**：验证节点的 TCB 状态是否最新

## 10. 测试策略

### 10.1 单元测试

```go
// 预编译合约测试
func TestSGXKeyCreate(t *testing.T) {
    // 测试密钥创建
}

func TestSGXSign(t *testing.T) {
    // 测试签名
}

func TestSGXECDH(t *testing.T) {
    // 测试 ECDH
}
```

### 10.2 集成测试

```go
// 多节点测试
func TestMultiNodeConsensus(t *testing.T) {
    // 启动多个节点
    // 验证共识达成
    // 验证数据一致性
}

func TestNodeJoin(t *testing.T) {
    // 测试新节点加入
    // 验证 SGX 远程证明
}
```

### 10.3 安全测试

```go
// 权限测试
func TestKeyOwnershipEnforcement(t *testing.T) {
    // 测试非所有者无法使用私钥
}

// 证明测试
func TestInvalidQuoteRejection(t *testing.T) {
    // 测试无效 Quote 被拒绝
}
```

## 11. 部署指南

### 11.1 硬件要求

- Intel CPU with SGX support (SGX2 recommended)
- 至少 16GB EPC (Enclave Page Cache)
- 支持 DCAP 的 SGX 驱动

### 11.2 软件要求

- Ubuntu 22.04 LTS
- Intel SGX SDK 2.19+
- Intel SGX DCAP 1.16+
- Gramine 1.5+
- Go 1.21+

### 11.3 部署步骤

```bash
# 1. 安装 SGX 驱动和 SDK
sudo apt install -y sgx-aesm-service libsgx-dcap-ql

# 2. 安装 Gramine
sudo apt install -y gramine

# 3. 构建 X Chain
cd go-ethereum
make geth

# 4. 生成 Gramine 签名
gramine-sgx-sign --manifest geth.manifest --output geth.manifest.sgx

# 5. 启动节点
./start-x-chain.sh
```

## 12. 硬件抽象层 (HAL)

### 12.1 设计目标

X Chain 默认使用 Intel SGX，但架构设计支持未来扩展到其他满足**恶意模型 (Malicious Model)** 的可信执行环境硬件。

### 12.2 安全模型要求

#### 12.2.1 恶意模型 (Malicious Model)

X Chain 要求底层硬件必须满足恶意模型，即：

- **不信任任何人**：包括云服务商、系统管理员、特权软件
- **只信任硬件本身**：安全性完全依赖硬件的密码学保证
- **抵抗特权攻击**：即使攻击者拥有 root 权限或物理访问权限，也无法窃取 enclave 内的秘密

#### 12.2.2 硬件分类

| 硬件 | 安全模型 | 是否支持 | 原因 |
|------|----------|----------|------|
| Intel SGX | 恶意模型 | 支持（默认） | 不信任 OS/Hypervisor，硬件级隔离 |
| Intel TDX | 恶意模型 | 未来支持 | VM 级 TEE，不信任 Hypervisor |
| RISC-V Keystone | 恶意模型 | 未来支持 | 开源 TEE，硬件级隔离 |
| ARM TrustZone | 半诚实模型 | 不支持 | 信任 Secure World 特权软件 |
| AMD SEV/SEV-SNP | 半诚实模型 | 不支持 | 信任 AMD 固件，内存加密但无完整性保护 |
| AWS Nitro Enclaves | 半诚实模型 | 不支持 | 信任 AWS Hypervisor |

### 12.3 硬件抽象层接口

```go
// internal/tee/hal.go
package tee

// TEEType 定义支持的 TEE 类型
type TEEType uint8

const (
    TEE_SGX      TEEType = 0x01  // Intel SGX (默认)
    TEE_TDX      TEEType = 0x02  // Intel TDX (未来)
    TEE_KEYSTONE TEEType = 0x03  // RISC-V Keystone (未来)
)

// TEEProvider 是硬件抽象层的核心接口
// 任何新的 TEE 硬件都必须实现此接口
type TEEProvider interface {
    // 基本信息
    Type() TEEType
    Name() string
    
    // 远程证明
    GenerateQuote(reportData []byte) ([]byte, error)
    VerifyQuote(quote []byte) (*QuoteVerificationResult, error)
    
    // 证书生成 (用于 RA-TLS)
    GenerateCertificate(privateKey crypto.PrivateKey) (*x509.Certificate, error)
    VerifyCertificate(cert *x509.Certificate) error
    
    // 代码度量
    GetCodeMeasurement() ([]byte, error)      // 类似 MRENCLAVE
    GetSignerMeasurement() ([]byte, error)    // 类似 MRSIGNER
    
    // 数据密封
    Seal(data []byte, policy SealPolicy) ([]byte, error)
    Unseal(sealedData []byte) ([]byte, error)
    
    // 硬件随机数
    GetRandomBytes(length int) ([]byte, error)
    
    // 安全模型验证
    SecurityModel() SecurityModel
    ValidateMaliciousModel() error  // 验证是否满足恶意模型
}

// SecurityModel 定义安全模型类型
type SecurityModel uint8

const (
    MODEL_MALICIOUS    SecurityModel = 0x01  // 恶意模型 (必需)
    MODEL_SEMI_HONEST  SecurityModel = 0x02  // 半诚实模型 (不支持)
)

// SealPolicy 定义数据密封策略
type SealPolicy uint8

const (
    SEAL_TO_ENCLAVE SealPolicy = 0x01  // 密封到特定 enclave (MRENCLAVE)
    SEAL_TO_SIGNER  SealPolicy = 0x02  // 密封到签名者 (MRSIGNER)
)

// QuoteVerificationResult 包含 Quote 验证结果
type QuoteVerificationResult struct {
    Valid           bool
    CodeMeasurement []byte
    SignerMeasurement []byte
    TCBStatus       TCBStatus
    Timestamp       time.Time
    AdditionalData  map[string]interface{}
}

// TCBStatus 定义 TCB 状态
type TCBStatus uint8

const (
    TCB_UP_TO_DATE      TCBStatus = 0x00
    TCB_OUT_OF_DATE     TCBStatus = 0x01
    TCB_REVOKED         TCBStatus = 0x02
    TCB_CONFIGURATION_NEEDED TCBStatus = 0x03
)
```

### 12.4 SGX 实现

```go
// internal/tee/sgx/provider.go
package sgx

type SGXProvider struct {
    dcapClient *DCAPClient
    config     *SGXConfig
}

func NewSGXProvider(config *SGXConfig) (*SGXProvider, error) {
    // 验证 SGX 可用性
    if !isSGXAvailable() {
        return nil, ErrSGXNotAvailable
    }
    
    return &SGXProvider{
        dcapClient: NewDCAPClient(),
        config:     config,
    }, nil
}

func (p *SGXProvider) Type() TEEType {
    return TEE_SGX
}

func (p *SGXProvider) Name() string {
    return "Intel SGX"
}

func (p *SGXProvider) SecurityModel() SecurityModel {
    return MODEL_MALICIOUS
}

func (p *SGXProvider) ValidateMaliciousModel() error {
    // SGX 满足恶意模型，直接返回 nil
    return nil
}

func (p *SGXProvider) GenerateQuote(reportData []byte) ([]byte, error) {
    // 通过 Gramine 的 /dev/attestation 接口生成 Quote
    // 1. 写入 report_data
    if err := os.WriteFile("/dev/attestation/user_report_data", reportData, 0600); err != nil {
        return nil, err
    }
    
    // 2. 读取 Quote
    quote, err := os.ReadFile("/dev/attestation/quote")
    if err != nil {
        return nil, err
    }
    
    return quote, nil
}

func (p *SGXProvider) VerifyQuote(quote []byte) (*QuoteVerificationResult, error) {
    // 使用 DCAP 验证 Quote
    return p.dcapClient.VerifyQuote(quote)
}

func (p *SGXProvider) GetRandomBytes(length int) ([]byte, error) {
    // 使用 RDRAND 指令获取硬件随机数
    buf := make([]byte, length)
    if _, err := rand.Read(buf); err != nil {
        return nil, err
    }
    return buf, nil
}
```

### 12.5 未来硬件扩展指南

当需要支持新的 TEE 硬件时，必须：

1. **验证安全模型**：确认硬件满足恶意模型要求
2. **实现 TEEProvider 接口**：实现所有必需的方法
3. **添加 TEEType 常量**：在 `TEEType` 中添加新的硬件类型
4. **实现远程证明**：提供 Quote 生成和验证功能
5. **实现数据密封**：提供与 SGX sealing 等效的功能
6. **测试验证**：通过所有安全测试

```go
// 示例：未来 Intel TDX 实现
// internal/tee/tdx/provider.go
package tdx

type TDXProvider struct {
    // TDX 特定配置
}

func (p *TDXProvider) Type() TEEType {
    return TEE_TDX
}

func (p *TDXProvider) SecurityModel() SecurityModel {
    return MODEL_MALICIOUS  // TDX 满足恶意模型
}

func (p *TDXProvider) ValidateMaliciousModel() error {
    // TDX 满足恶意模型
    return nil
}

// ... 实现其他接口方法
```

### 12.6 运行时硬件检测

```go
// internal/tee/detect.go
package tee

// DetectTEE 自动检测可用的 TEE 硬件
func DetectTEE() (TEEProvider, error) {
    // 优先检测 SGX
    if isSGXAvailable() {
        provider, err := sgx.NewSGXProvider(nil)
        if err == nil {
            return provider, nil
        }
    }
    
    // 未来：检测 TDX
    // if isTDXAvailable() {
    //     return tdx.NewTDXProvider(nil)
    // }
    
    // 未来：检测 Keystone
    // if isKeystoneAvailable() {
    //     return keystone.NewKeystoneProvider(nil)
    // }
    
    return nil, ErrNoTEEAvailable
}

// ValidateTEEProvider 验证 TEE 提供者是否满足要求
func ValidateTEEProvider(provider TEEProvider) error {
    // 1. 验证安全模型
    if provider.SecurityModel() != MODEL_MALICIOUS {
        return ErrNotMaliciousModel
    }
    
    // 2. 验证恶意模型实现
    if err := provider.ValidateMaliciousModel(); err != nil {
        return err
    }
    
    return nil
}
```

### 12.7 配置文件

```toml
# config.toml

[tee]
# 默认使用 SGX，未来可配置为其他满足恶意模型的硬件
type = "sgx"  # 可选值: "sgx", "tdx", "keystone"

# 是否强制验证恶意模型
require_malicious_model = true

[tee.sgx]
# SGX 特定配置
dcap_url = "https://api.trustedservices.intel.com/sgx/certification/v4"
allowed_tcb_status = ["UpToDate", "SWHardeningNeeded"]

# [tee.tdx]
# TDX 特定配置 (未来)

# [tee.keystone]
# Keystone 特定配置 (未来)
```

## 13. 附录

### 13.1 参考资料

- [Intel SGX Developer Reference](https://download.01.org/intel-sgx/sgx-linux/2.19/docs/)
- [Gramine Documentation](https://gramine.readthedocs.io/)
- [go-ethereum Documentation](https://geth.ethereum.org/docs)
- [RA-TLS Specification](https://gramine.readthedocs.io/en/stable/attestation.html)

### 12.2 术语表

| 术语 | 定义 |
|------|------|
| SGX | Intel Software Guard Extensions，硬件可信执行环境 |
| Enclave | SGX 保护的内存区域 |
| MRENCLAVE | Enclave 代码和数据的度量值 |
| MRSIGNER | Enclave 签名者的度量值 |
| Quote | SGX 远程证明数据结构 |
| RA-TLS | Remote Attestation TLS，带远程证明的 TLS |
| DCAP | Data Center Attestation Primitives |
| Sealing | SGX 数据持久化加密机制 |
| TCB | Trusted Computing Base |
