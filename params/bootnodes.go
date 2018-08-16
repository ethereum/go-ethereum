// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	// Ethereum Foundation Go Bootnodes
	"enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303", // IE
	"enode://3f1d12044546b76342d59d4a05532c14b85aa669704bfe1f864fe079415aa2c02d743e03218e57a33fb94523adb54032871a6c51b2cc5514cb7c7e35b3ed0a99@13.93.211.84:30303",  // US-WEST
	"enode://78de8a0916848093c73790ead81d1928bec737d565119932b98c6b100d944b7a95e94f847f689fc723399d2e31129d182f7ef3863f2b4c820abbf3ab2722344d@191.235.84.50:30303", // BR
	"enode://158f8aab45f6d19c6cbf4a089c2670541a8da11978a2f90dbf6a502a4a3bab80d288afdbeb7ec0ef6d92de563767f3b1ea9e8e334ca711e9f8e2df5a0385e8e6@13.75.154.138:30303", // AU
	"enode://1118980bf48b0a3640bdba04e0fe78b1add18e1cd99bf22d53daac1fd9972ad650df52176e7c7d89d1114cfef2bc23a2959aa54998a46afcf7d91809f0855082@52.74.57.123:30303",  // SG

	// Ethereum Foundation C++ Bootnodes
	"enode://979b7fa28feeb35a4741660a16076f1943202cb72b6af70d327f053e248bab9ba81760f39d0701ef1d8f89cc1fbd2cacba0710a12cd5314d5e0c9021aa3637f9@5.1.83.226:30303", // DE
}

// ClassicBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the Ethereum Classic network.
var ClassicBootnodes = []string{
	"enode://e809c4a2fec7daed400e5e28564e23693b23b2cc5a019b612505631bbe7b9ccf709c1796d2a3d29ef2b045f210caf51e3c4f5b6d3587d43ad5d6397526fa6179@174.112.32.157:30303",
	"enode://6e538e7c1280f0a31ff08b382db5302480f775480b8e68f8febca0ceff81e4b19153c6f8bf60313b93bef2cc34d34e1df41317de0ce613a201d1660a788a03e2@52.206.67.235:30303",
	"enode://5fbfb426fbb46f8b8c1bd3dd140f5b511da558cd37d60844b525909ab82e13a25ee722293c829e52cb65c2305b1637fa9a2ea4d6634a224d5f400bfe244ac0de@162.243.55.45:30303",
	"enode://42d8f29d1db5f4b2947cd5c3d76c6d0d3697e6b9b3430c3d41e46b4bb77655433aeedc25d4b4ea9d8214b6a43008ba67199374a9b53633301bca0cd20c6928ab@104.155.176.151:30303",
	"enode://814920f1ec9510aa9ea1c8f79d8b6e6a462045f09caa2ae4055b0f34f7416fca6facd3dd45f1cf1673c0209e0503f02776b8ff94020e98b6679a0dc561b4eba0@104.154.136.117:30303",
	"enode://72e445f4e89c0f476d404bc40478b0df83a5b500d2d2e850e08eb1af0cd464ab86db6160d0fde64bd77d5f0d33507ae19035671b3c74fec126d6e28787669740@104.198.71.200:30303",
	"enode://5cd218959f8263bc3721d7789070806b0adff1a0ed3f95ec886fb469f9362c7507e3b32b256550b9a7964a23a938e8d42d45a0c34b332bfebc54b29081e83b93@35.187.57.94:30303",
	"enode://39abab9d2a41f53298c0c9dc6bbca57b0840c3ba9dccf42aa27316addc1b7e56ade32a0a9f7f52d6c5db4fe74d8824bcedfeaecf1a4e533cacb71cf8100a9442@144.76.238.49:30303",
	"enode://f50e675a34f471af2438b921914b5f06499c7438f3146f6b8936f1faeb50b8a91d0d0c24fb05a66f05865cd58c24da3e664d0def806172ddd0d4c5bdbf37747e@144.76.238.49:30306",
	"enode://6dd3ac8147fa82e46837ec8c3223d69ac24bcdbab04b036a3705c14f3a02e968f7f1adfcdb002aacec2db46e625c04bf8b5a1f85bb2d40a479b3cc9d45a444af@104.237.131.102:30303",
}

// SocialBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the Ethereum Social network.
var SocialBootnodes = []string{
	"enode://38a3bdd683008f2b404fbd8e59a4ae7377fb1b796be8aca02861a6864304df7f443ae9669d0072d567eb30ab2183556a3cd832b8f2c99246e9a3d9f64ecdc1af@52.78.243.91:30303",
	"enode://ee31120190438ca3842afccca9d732d8bfca4bbf9b846fd2bb11178194aa49a74d77ff4801d50a7bd9eb3629f8903661d0fb973e7f43b395263530b390002033@13.209.99.197:30303",
	"enode://d538165bf6026602ba9ac296b2b56994e03bb917c73b79cbb11df75a45576fa74df494097bdbcda9bf2c0954a47a65b65674780fa1fbde5bcc89a34870d44983@13.125.206.82:30303",
	"enode://67b5de9a4562ba0a01877e3876249c8e551844424773bdbf9713d126b3f144ac7a49d8eb06fc9830871f03b50a7d9b5d98d9d1be5544aef8afcaa10eea2fb9eb@13.125.68.29:30303",
	"enode://2d31dd1f8acd956cf36a1c3f27e374f5b94c55df4206749b03a6d0a50366c8090280c91f71aad00886cbde6ebbfcabeaaa91bd910b16e4fb398b337e9ecfdbd9@13.125.232.71:30303",
}

// CallistoBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the Callisto network.
var CallistoBootnodes = []string{
	"enode://ab0d9fe81f23653c3217303d3a4bc035be908b05f8b12d6f155ab4598d7760cfea1e009e414771279c9ffc3499950283afaabe099a5329c5a6013f57d77de0a6@104.236.96.118:30303",
	"enode://9e2d9dc2639e02893aa17c80e6ba8e8803fd3166083b622a841fc852161112720281514436f7605c89041d5efa1738215185c4c4024ff812b0f500c403cc0ab1@206.189.47.198:30303",
	"enode://149ba679e8851c3e0d030e0dc0336984b97c83ef649e68ec113dcf266449364cc1ec8ee27950f71b00c2182ef504894fa7bff19f6741978ced67e9e4b6536d2a@206.189.45.31:30303",
	"enode://2038f895f0870f76456a879b97a276135418295ff9f6e24b6de11225360adc87d3aa6d90302aafb36a4cd9542eaf0f160ae48439da938d7e3031d838ef11b8e8@13.125.216.165:30303", // clopool.net
	"enode://b292d40b86dc73b42b0da20b146957e5a06a4f09b2d77b26d07ca34a0e7b5f06972e1b589722b1b3591c8126b61d40d88a1ee32990451e2baa9df563d34668e4@13.125.242.67:30303",  // clopool.net
	"enode://ae7b2bfa18c137b8f4e4a3f1485d734b40198531916bcf59637bc09ad6b4e310d1077d9bd8f5aa7d8271b97673e28595e73d81b7b8c9a862c78f11fb8aa4f6a0@13.125.194.99:30303",  // clopool.net
	"enode://09fdeea69fce23dd021d9db2930e508c1a0741c56a0ae7efceb0203a562838412dd389ff94bc3b5b888eec7a16bd1960734efc551ffbb95ad72b698a9105e5d4@13.209.49.184:30303",  // clopool.net
	"enode://8285433aecaf42dc9372bff4d44f119cc58db93b5caa169f8763db777112cd55a2d6a18370560cfad8b78196fe8c9f39ec7e30bdba274bc74c11f152028e6980@52.78.148.64:30303",   // clopool.net
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	"enode://30b7ab30a01c124a6cceca36863ece12c4f5fa68e3ba9b0b51407ccc002eeed3b3102d20a88f1c1d3c3154e2449317b8ef95090e77b312d5cc39354f86d5d606@52.176.7.10:30303",    // US-Azure geth
	"enode://865a63255b3bb68023b6bffd5095118fcc13e79dcf014fe4e47e065c350c7cc72af2e53eff895f11ba1bbb6a2b33271c1116ee870f266618eadfc2e78aa7349c@52.176.100.77:30303",  // US-Azure parity
	"enode://6332792c4a00e3e4ee0926ed89e0d27ef985424d97b6a45bf0f23e51f0dcb5e66b875777506458aea7af6f9e4ffb69f43f3778ee73c81ed9d34c51c4b16b0b0f@52.232.243.152:30303", // Parity
	"enode://94c15d1b9e2fe7ce56e458b9a3b672ef11894ddedd0c6f247e0f1d3487f52b66208fb4aeb8179fce6e3a749ea93ed147c37976d67af557508d199d9594c35f09@192.81.208.223:30303", // @gpip
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{
	"enode://a24ac7c5484ef4ed0c5eb2d36620ba4e4aa13b8c84684e1b4aab0cebea2ae45cb4d375b77eab56516d34bfbd3c1a833fc51296ff084b770b94fb9028c4d25ccf@52.169.42.101:30303", // IE
	"enode://343149e4feefa15d882d9fe4ac7d88f885bd05ebb735e547f12e12080a9fa07c8014ca6fd7f373123488102fe5e34111f8509cf0b7de3f5b44339c9f25e87cb8@52.3.158.184:30303",  // INFURA
	"enode://b6b28890b006743680c52e64e0d16db57f28124885595fa03a562be1d2bf0f3a1da297d56b13da25fb992888fd556d4c1a27b1f39d531bde7de1921c90061cc6@159.89.28.211:30303", // AKASHA
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	"enode://06051a5573c81934c9554ef2898eb13b33a34b94cf36b202b69fde139ca17a85051979867720d4bdae4323d4943ddf9aeeb6643633aa656e0be843659795007a@35.177.226.168:30303",
	"enode://0cc5f5ffb5d9098c8b8c62325f3797f56509bff942704687b6530992ac706e2cb946b90a34f1f19548cd3c7baccbcaea354531e5983c7d1bc0dee16ce4b6440b@40.118.3.223:30304",
	"enode://1c7a64d76c0334b0418c004af2f67c50e36a3be60b5e4790bdac0439d21603469a85fad36f2473c9a80eb043ae60936df905fa28f1ff614c3e5dc34f15dcd2dc@40.118.3.223:30306",
	"enode://85c85d7143ae8bb96924f2b54f1b3e70d8c4d367af305325d30a61385a432f247d2c75c45c6b4a60335060d072d7f5b35dd1d4c45f76941f62a4f83b6e75daaf@40.118.3.223:30307",
}
