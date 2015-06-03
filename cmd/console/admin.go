package main

/*
node admin bindings
*/

func (js *jsre) adminBindings() {

	js.re.Set("admin", struct{}{})
	t, _ := js.re.Get("admin")
	admin := t.Object()

	admin.Set("miner", struct{}{})
	t, _ = admin.Get("miner")
	miner := t.Object()
	miner.Set("start", js.startMining)
	//	miner.Set("stop", js.stopMining)
	//	miner.Set("hashrate", js.hashrate)
	//	miner.Set("setExtra", js.setExtra)
	//	miner.Set("setGasPrice", js.setGasPrice)
	//	miner.Set("startAutoDAG", js.startAutoDAG)
	//	miner.Set("stopAutoDAG", js.stopAutoDAG)
	//	miner.Set("makeDAG", js.makeDAG)

	/*
		ethO, _ := js.re.Get("eth")
		eth := ethO.Object()
		eth.Set("pendingTransactions", js.pendingTransactions)
		eth.Set("resend", js.resend)
		eth.Set("sign", js.sign)

		js.re.Set("admin", struct{}{})
		t, _ := js.re.Get("admin")
		admin := t.Object()
		admin.Set("addPeer", js.addPeer)
		admin.Set("startRPC", js.startRPC)
		admin.Set("stopRPC", js.stopRPC)
		admin.Set("nodeInfo", js.nodeInfo)
		admin.Set("peers", js.peers)
		admin.Set("newAccount", js.newAccount)
		admin.Set("unlock", js.unlock)
		admin.Set("import", js.importChain)
		admin.Set("export", js.exportChain)
		admin.Set("verbosity", js.verbosity)
		admin.Set("progress", js.downloadProgress)
		admin.Set("setSolc", js.setSolc)

		admin.Set("contractInfo", struct{}{})
		t, _ = admin.Get("contractInfo")
		cinfo := t.Object()
		// newRegistry officially not documented temporary option
		cinfo.Set("start", js.startNatSpec)
		cinfo.Set("stop", js.stopNatSpec)
		cinfo.Set("newRegistry", js.newRegistry)
		cinfo.Set("get", js.getContractInfo)
		cinfo.Set("register", js.register)
		cinfo.Set("registerUrl", js.registerUrl)
		// cinfo.Set("verify", js.verify)



		admin.Set("debug", struct{}{})
		t, _ = admin.Get("debug")
		debug := t.Object()
		js.re.Set("sleep", js.sleep)
		debug.Set("backtrace", js.backtrace)
		debug.Set("printBlock", js.printBlock)
		debug.Set("dumpBlock", js.dumpBlock)
		debug.Set("getBlockRlp", js.getBlockRlp)
		debug.Set("setHead", js.setHead)
		debug.Set("processBlock", js.debugBlock)
		debug.Set("seedhash", js.seedHash)
		// undocumented temporary
		debug.Set("waitForBlocks", js.waitForBlocks)
	*/
}
