package resolver

const ( // built-in contracts address and code
	ContractCodeURLhint = "0x60c180600c6000396000f30060003560e060020a90048063300a3bbf14601557005b6024600435602435604435602a565b60006000f35b6000600084815260200190815260200160002054600160a060020a0316600014806078575033600160a060020a03166000600085815260200190815260200160002054600160a060020a0316145b607f5760bc565b336000600085815260200190815260200160002081905550806001600085815260200190815260200160002083610100811060b657005b01819055505b50505056"
	/*
		contract URLhint {
			function register(uint256 _hash, uint8 idx, uint256 _url) {
				if (owner[_hash] == 0 || owner[_hash] == msg.sender) {
					owner[_hash] = msg.sender;
					url[_hash][idx] = _url;
				}
			}
			mapping (uint256 => address) owner;
			mapping (uint256 => uint256[256]) url;
		}
	*/

	ContractCodeHashReg = "0x609880600c6000396000f30060003560e060020a9004806331e12c2014601f578063d66d6c1014602b57005b6025603d565b60006000f35b6037600435602435605d565b60006000f35b600054600160a060020a0316600014605357605b565b336000819055505b565b600054600160a060020a031633600160a060020a031614607b576094565b8060016000848152602001908152602001600020819055505b505056"
	/*
		   contract HashReg {
			function setowner() {
				if (owner == 0) {
					owner = msg.sender;
				}
			}
		   	function register(uint256 _key, uint256 _content) {
				if (msg.sender == owner) {
		   			content[_key] = _content;
				}
		   	}
			address owner;
		   	mapping (uint256 => uint256) content;
		   }
	*/

	GlobalRegistrar     = `var GlobalRegistrar = web3.eth.contract([{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"name","outputs":[{"name":"o_name","type":"bytes32"}],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"owner","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"content","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"addr","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"}],"name":"reserve","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"subRegistrar","outputs":[{"name":"o_subRegistrar","type":"address"}],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"},{"name":"_newOwner","type":"address"}],"name":"transfer","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"},{"name":"_registrar","type":"address"}],"name":"setSubRegistrar","outputs":[],"type":"function"},{"constant":false,"inputs":[],"name":"Registrar","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"},{"name":"_a","type":"address"},{"name":"_primary","type":"bool"}],"name":"setAddress","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"},{"name":"_content","type":"bytes32"}],"name":"setContent","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"}],"name":"disown","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"register","outputs":[{"name":"","type":"address"}],"type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"name","type":"bytes32"}],"name":"Changed","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"name","type":"bytes32"},{"indexed":true,"name":"addr","type":"address"}],"name":"PrimaryChanged","type":"event"}]);`
	GlobalRegistrarAddr = "0xc6d9d2cd449a754c494264e1809c50e34d64562b"
)
