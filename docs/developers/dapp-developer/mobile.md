---
title: Geth for Mobile
description: Introduction to mobile development with Geth
---

<Note>
Geth [no longer publishes](https://github.com/ethereum/go-ethereum/pull/26599) builds for mobile.
</Note>

In the past, Geth was released for Android and IoS to support embedding clients into mobile applications. However, the move to proof-of-stake based consensus introduced the need for a consensus client to be run alongside Geth in order to track the head of the blockchain, breaking the ability for Geth light clients to run on a mobile device and handle API requests from mobile apps.

Supporting mobile app development is no longer part of Geth's remit but it remains possible for other teams to devise ways to build on Ethereum in a mobile environment.