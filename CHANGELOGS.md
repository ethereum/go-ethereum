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
