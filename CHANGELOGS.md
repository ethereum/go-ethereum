# CHANGELOGS

## 2022-05-04

Tag: None.

Current rev: a79e72f69701695185f2f71788d17998bdd5a5a8.

Based on https://github.com/ethereum/go-ethereum v1.10.13.

**Notable changes:**

### 1. Disable consensus and p2p service.

Related commits:

+ [a4eb31f4d2959a4ea97edf38aa13e4f87a81b1a1](https://github.com/scroll-tech/go-ethereum/commit/a4eb31f4d2959a4ea97edf38aa13e4f87a81b1a1) (PR [#8](https://github.com/scroll-tech/go-ethereum/pull/8))
+ [8b16b4cefae82f7d3f73a37df9e04d7d356a7a23](https://github.com/scroll-tech/go-ethereum/commit/8b16b4cefae82f7d3f73a37df9e04d7d356a7a23) (PR [#29](https://github.com/scroll-tech/go-ethereum/pull/29))

### 2. Add more detailed execution trace for zkevm-circuits proving.

Related commits:

+ [7745fd584018eb0ef63db1b17e27b968c9ae5dca](https://github.com/scroll-tech/go-ethereum/commit/7745fd584018eb0ef63db1b17e27b968c9ae5dca) (PR [#19](https://github.com/scroll-tech/go-ethereum/pull/19))
+ [69c291cf7ac43be7e10979457e4137af0c63ce0e](https://github.com/scroll-tech/go-ethereum/commit/69c291cf7ac43be7e10979457e4137af0c63ce0e) (PR [#20](https://github.com/scroll-tech/go-ethereum/pull/20))
+ [a5999cee905c16a24abb4580f43dca30ac9441d5](https://github.com/scroll-tech/go-ethereum/commit/a5999cee905c16a24abb4580f43dca30ac9441d5) (PR [#44](https://github.com/scroll-tech/go-ethereum/pull/44))
+ [13ea5c234b359e9eebe92b816ae6f02e876d6add](https://github.com/scroll-tech/go-ethereum/commit/13ea5c234b359e9eebe92b816ae6f02e876d6add) (PR [#46](https://github.com/scroll-tech/go-ethereum/pull/46))
+ [51281549254bfbfd3ad1dd87ffc9604c2b2e5a77](https://github.com/scroll-tech/go-ethereum/commit/51281549254bfbfd3ad1dd87ffc9604c2b2e5a77) (PR [#56](https://github.com/scroll-tech/go-ethereum/pull/56))
+ [8ccc10541dd49d52c70230cc0f8210d5da64248b](https://github.com/scroll-tech/go-ethereum/commit/8ccc10541dd49d52c70230cc0f8210d5da64248b) (PR [#58](https://github.com/scroll-tech/go-ethereum/pull/58))
+ [360115e61fb33aee60e02d7a907a2f0e79935ee2](https://github.com/scroll-tech/go-ethereum/commit/360115e61fb33aee60e02d7a907a2f0e79935ee2)
+ [9f1d8552e4d0abb30096ab3a75d9ab4616ac23b6](https://github.com/scroll-tech/go-ethereum/commit/9f1d8552e4d0abb30096ab3a75d9ab4616ac23b6) (PR [#66](https://github.com/scroll-tech/go-ethereum/pull/66))
+ [2329d324098f799e4d547d6ef14a77f52fc469cf](https://github.com/scroll-tech/go-ethereum/commit/2329d324098f799e4d547d6ef14a77f52fc469cf) (PR [#71](https://github.com/scroll-tech/go-ethereum/pull/71))

(

And some fixes regarding encoding:

+ [33fcd2bf6d4fa467bd8207bb1dc9c55bbed6be9b](https://github.com/scroll-tech/go-ethereum/commit/33fcd2bf6d4fa467bd8207bb1dc9c55bbed6be9b) (PR [#72](https://github.com/scroll-tech/go-ethereum/pull/72))
+ [3d3c9d3edff7cc6c3445f4fc9cf072df7853ee7d](https://github.com/scroll-tech/go-ethereum/commit/3d3c9d3edff7cc6c3445f4fc9cf072df7853ee7d) (PR [#74](https://github.com/scroll-tech/go-ethereum/pull/74))
+ [06190d0642afe076b257d67ad40b6b85e0a6e087](https://github.com/scroll-tech/go-ethereum/commit/06190d0642afe076b257d67ad40b6b85e0a6e087) (PR [#75](https://github.com/scroll-tech/go-ethereum/pull/75))

)


### 3. Optimization to reduce GC pressure

Related commits:

+ [09a31ccc66bcf676f71451bb1f3fde2e44849da3](https://github.com/scroll-tech/go-ethereum/commit/09a31ccc66bcf676f71451bb1f3fde2e44849da3) (PR [#43](https://github.com/scroll-tech/go-ethereum/pull/43))
+ [a79e72f69701695185f2f71788d17998bdd5a5a8](https://github.com/scroll-tech/go-ethereum/commit/a79e72f69701695185f2f71788d17998bdd5a5a8) (PR [#83](https://github.com/scroll-tech/go-ethereum/pull/83))

### 4. Misc

4.1 enable London fork rules from the beginning

Related commits:

+ [c180aa2e75d80dda90719b58690111b0d5b69f21](https://github.com/scroll-tech/go-ethereum/commit/c180aa2e75d80dda90719b58690111b0d5b69f21) (PR [#76](https://github.com/scroll-tech/go-ethereum/pull/76))

## 2022-06-27

Tag: None.

Current rev: c516a9e47739bee96e70aced01fad255c0311897.

Based on https://github.com/ethereum/go-ethereum v1.10.13.

**Notable changes:**

### 1. Add zktrie, allow switch trie type by config.

Related commits:

+ [c516a9e47739bee96e70aced01fad255c0311897](https://github.com/scroll-tech/go-ethereum/commit/c516a9e47739bee96e70aced01fad255c0311897) (PR [#113](https://github.com/scroll-tech/go-ethereum/pull/113))

### 2. Add more detailed execution trace for zkevm-circuits proving.

Related commits:

+ [d3bc8322dc503fa1b927a60b518f0b195641ffdf](https://github.com/scroll-tech/go-ethereum/commit/d3bc8322dc503fa1b927a60b518f0b195641ffdf) (PR [#102](https://github.com/scroll-tech/go-ethereum/pull/102))

(

Fields change:

+ [571dcad4be512225bb1209f8008a8577eab29ded](https://github.com/scroll-tech/go-ethereum/commit/571dcad4be512225bb1209f8008a8577eab29ded) (PR [#98](https://github.com/scroll-tech/go-ethereum/pull/98))
+ [e15d0d35cba2aa6aab932df2691e7544e4ffda78](https://github.com/scroll-tech/go-ethereum/commit/e15d0d35cba2aa6aab932df2691e7544e4ffda78) (PR [#117](https://github.com/scroll-tech/go-ethereum/pull/117))

Bug fix:

+ [f73142728206ddc4b89d3b3e9b5549933eba94fe](https://github.com/scroll-tech/go-ethereum/commit/f73142728206ddc4b89d3b3e9b5549933eba94fe) (PR [#119](https://github.com/scroll-tech/go-ethereum/pull/119))

)


### 3. Increase tps or reduce GC pressure

Related commits:

+ [9199413d21c6c08f14ff968c472206e5ebff0518](https://github.com/scroll-tech/go-ethereum/commit/9199413d21c6c08f14ff968c472206e5ebff0518) (PR [#92](https://github.com/scroll-tech/go-ethereum/pull/92))
+ [9b99f2e17425fa16d1835cbfe47f9015321faae1](https://github.com/scroll-tech/go-ethereum/commit/9b99f2e17425fa16d1835cbfe47f9015321faae1) (PR [#104](https://github.com/scroll-tech/go-ethereum/pull/104))

### 4. Misc

4.1 opcode operation

Related commits:

+ [21b65f4944667e574c29f37db6da7185b7dfa444](https://github.com/scroll-tech/go-ethereum/commit/21b65f4944667e574c29f37db6da7185b7dfa444) (PR [#118](https://github.com/scroll-tech/go-ethereum/pull/118))

4.2 The changes of module import

Related commits:

+ [9199413d21c6c08f14ff968c472206e5ebff0518](https://github.com/scroll-tech/go-ethereum/commit/9199413d21c6c08f14ff968c472206e5ebff0518) (PR [#92](https://github.com/scroll-tech/go-ethereum/pull/92))

4.3 The changes of ci、jenkins、docker、makefile and readme

Related commits:

+ [35f6a91cd5d5bd2ecfc865f6c0c0b239727f55ee](https://github.com/scroll-tech/go-ethereum/commit/35f6a91cd5d5bd2ecfc865f6c0c0b239727f55ee) (PR [#111](https://github.com/scroll-tech/go-ethereum/pull/111))
+ [3410a56d866735f6a81eb8e5bae3976751ab0691](https://github.com/scroll-tech/go-ethereum/commit/3410a56d866735f6a81eb8e5bae3976751ab0691) (PR [#121](https://github.com/scroll-tech/go-ethereum/pull/121))

## 2022-07-30

Tag: None.

Current rev: d421337df58074bdee8d8cb8fa592ece5a2300e8.

Based on https://github.com/ethereum/go-ethereum v1.10.13.

**Notable changes:**

### 1. Disable memory trace

Related commits:

+ [d421337df58074bdee8d8cb8fa592ece5a2300e8](https://github.com/scroll-tech/go-ethereum/commit/d421337df58074bdee8d8cb8fa592ece5a2300e8) (PR [#134](https://github.com/scroll-tech/go-ethereum/pull/134))

### 2. Add more opcode handlings

Related commits:

+ [eb11a84c56b30bf7e2345db9b7532542336a5581](https://github.com/scroll-tech/go-ethereum/commit/eb11a84c56b30bf7e2345db9b7532542336a5581) (PR [#128](https://github.com/scroll-tech/go-ethereum/pull/128))

### 3. Include zktrie witness in block trace; add demo for generating witness data for mpt circuit

Related commits:

+ [3682e05f3f2495af437234e2036412e9f7ed51b7](https://github.com/scroll-tech/go-ethereum/commit/3682e05f3f2495af437234e2036412e9f7ed51b7) (PR [#123](https://github.com/scroll-tech/go-ethereum/pull/123))

(

Fields change:

+ [f9952a396fb558fe1a3f5804f66a8a1683cd44d6](https://github.com/scroll-tech/go-ethereum/commit/f9952a396fb558fe1a3f5804f66a8a1683cd44d6) (PR [#133](https://github.com/scroll-tech/go-ethereum/pull/133))

Bug fix:

+ [fefa8b99c7b3dea8f15e8350245f71c3bbafa046](https://github.com/scroll-tech/go-ethereum/commit/fefa8b99c7b3dea8f15e8350245f71c3bbafa046) (PR [#132](https://github.com/scroll-tech/go-ethereum/pull/132))
+ [37dbb86aa615ba1ab583946f084b0ce190975478](https://github.com/scroll-tech/go-ethereum/commit/37dbb86aa615ba1ab583946f084b0ce190975478) (PR [#126](https://github.com/scroll-tech/go-ethereum/pull/126))

)