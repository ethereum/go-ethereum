function HandleMessage(data) {
	var message;
	try { message = JSON.parse(data) } catch(e) {};

	if(message) {
		switch(message.type) {
			case "coinbase":
				return eth.coinBase();
			case "block":
				return eth.blockByNumber(0);
		}
	}
}
