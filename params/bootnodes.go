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

import "github.com/ethereum/go-ethereum/common"

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	// Ethereum Foundation Go Bootnodes
	"enode://d860a01f9722d78051619d1e2351aba3f43f943f6f00718d1b9baa4101932a1f5011f16bb2b1bb35db20d6fe28fa0bf09636d26a87d31de9ec6203eeedb1f666@18.138.108.67:30303", // bootnode-aws-ap-southeast-1-001
	"enode://22a8232c3abc76a16ae9d6c3b164f98775fe226f0917b0ca871128a74a8e9630b458460865bab457221f1d448dd9791d24c4e5d88786180ac185df813a68d4de@3.209.45.79:30303",   // bootnode-aws-us-east-1-001
	"enode://2b252ab6a1d0f971d9722cb839a42cb81db019ba44c08754628ab4a823487071b5695317c8ccd085219c3a03af063495b2f1da8d18218da2d6a82981b45e6ffc@65.108.70.101:30303", // bootnode-hetzner-hel
	"enode://4aeb4ab6c14b23e2c4cfdce879c04b0748a20d8e9b59e25ded2a08143e265c6c25936e74cbc8e641e3312ca288673d91f2f93f8e277de3cfa444ecdaaf982052@157.90.35.166:30303", // bootnode-hetzner-fsn
}

// HoodiBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Hoodi test network.
var HoodiBootnodes = []string{
	// EF DevOps
	"enode://2112dd3839dd752813d4df7f40936f06829fc54c0e051a93967c26e5f5d27d99d886b57b4ffcc3c475e930ec9e79c56ef1dbb7d86ca5ee83a9d2ccf36e5c240c@134.209.138.84:30303",
	"enode://60203fcb3524e07c5df60a14ae1c9c5b24023ea5d47463dfae051d2c9f3219f309657537576090ca0ae641f73d419f53d8e8000d7a464319d4784acd7d2abc41@209.38.124.160:30303",
	"enode://8ae4a48101b2299597341263da0deb47cc38aa4d3ef4b7430b897d49bfa10eb1ccfe1655679b1ed46928ef177fbf21b86837bd724400196c508427a6f41602cd@134.199.184.23:30303",
}

// HoleskyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Holesky test network.
var HoleskyBootnodes = []string{
	// EF DevOps
	"enode://ac906289e4b7f12df423d654c5a962b6ebe5b3a74cc9e06292a85221f9a64a6f1cfdd6b714ed6dacef51578f92b34c60ee91e9ede9c7f8fadc4d347326d95e2b@146.190.13.128:30303",
	"enode://a3435a0155a3e837c02f5e7f5662a2f1fbc25b48e4dc232016e1c51b544cb5b4510ef633ea3278c0e970fa8ad8141e2d4d0f9f95456c537ff05fdf9b31c15072@178.128.136.233:30303",
}

// SepoliaBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Sepolia test network.
var SepoliaBootnodes = []string{
	// EF DevOps
	"enode://4e5e92199ee224a01932a377160aa432f31d0b351f84ab413a8e0a42f4f36476f8fb1cbe914af0d9aef0d51665c214cf653c651c4bbd9d5550a934f241f1682b@138.197.51.181:30303", // sepolia-bootnode-1-nyc3
	"enode://143e11fb766781d22d92a2e33f8f104cddae4411a122295ed1fdb6638de96a6ce65f5b7c964ba3763bba27961738fef7d3ecc739268f3e5e771fb4c87b6234ba@146.190.1.103:30303",  // sepolia-bootnode-1-sfo3
	"enode://8b61dc2d06c3f96fddcbebb0efb29d60d3598650275dc469c22229d3e5620369b0d3dedafd929835fe7f489618f19f456fe7c0df572bf2d914a9f4e006f783a9@170.64.250.88:30303",  // sepolia-bootnode-1-syd1
	"enode://10d62eff032205fcef19497f35ca8477bea0eadfff6d769a147e895d8b2b8f8ae6341630c645c30f5df6e67547c03494ced3d9c5764e8622a26587b083b028e8@139.59.49.206:30303",  // sepolia-bootnode-1-blr1
	"enode://9e9492e2e8836114cc75f5b929784f4f46c324ad01daf87d956f98b3b6c5fcba95524d6e5cf9861dc96a2c8a171ea7105bb554a197455058de185fa870970c7c@138.68.123.152:30303", // sepolia-bootnode-1-ams3
}

// BerachainBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the Berachain mainnet.
var BerachainBootnodes = []string{
	// South Korea
	"enode://0c5a4a3c0e81fce2974e4d317d88df783731183d534325e32e0fdf8f4b119d7889fa254d3a38890606ec300d744e2aa9c87099a4a032f5c94efe53f3fcdfecfe@34.64.176.79:30303",
	"enode://b6a3137d3a36ef37c4d31843775a9dc293f41bcbde33b6309c80b1771b6634827cd188285136a57474427bd8845adc2f6fe2e0b106bd58d14795b08910b9c326@34.64.181.70:30303",
	"enode://0b6633300614bc2b9749aee0cace7a091ec5348762aee7b1d195f7616d03a9409019d9bef336624bab72e0d069cd4cf0b0de6fbbf53f04f6b6e4c5b39c6bdca6@34.64.39.31:30303",
	"enode://552b001abebb5805fcd734ad367cd05d9078d18f23ec598d7165460fadcfc51116ad95c418f7ea9a141aa8cbc496c8bea3322b67a5de0d3380f11aab1a797513@34.64.183.158:30303",

	// Singapore
	"enode://5b037f66099d5ded86eb7e1619f6d06ceb15609e8cc345ced22a4772b06178004e1490a3cd32fd1222789de4c6e4021c2d648a3d750f6d5323e64b771bbd8de7@34.87.142.180:30303",
	"enode://846db253c53753d3ea1197aec296306dc84c25f3afdf142b65cb0fe0f984de55072daa3bbf05a9aea046a38a2292403137b6eafefd5646fcf62120b74e3b898d@34.142.170.110:30303",
	"enode://64b7f6ee9bcd942ad4949c70f2077627f078a057dfd930e6e904e12643d8952f5ae87c91e24559765393f244a72c9d5c011d7d5176e59191d38f315db85a20f5@34.126.161.16:30303",
	"enode://cf4d19bfb8ec507427ec882bac0bac85a0c8c9ddaa0ec91b773bb614e5e09d107cd9fbe323b96f62f31c493f8f42cc5495c18b87c08560c5dea1dfd25256dcf6@35.247.162.2:30303",

	// Germany
	"enode://bb7e44178543431feac8f0ee3827056b7b84d8235b802a8bdbbcd4939dab7f7dd2579ff577a38b002bb0139792af67abd2dd5c9f4f85b8da6e914fa76dca82bc@35.198.150.35:30303",
	"enode://8fef1f5df45e7b31be00a21e1da5665d5a5f5bf4c379086b843f03eade941bdd157f08c95b31880c492577edb9a9b185df7191eaebf54ab06d5bd683b289f3af@34.107.7.241:30303",
	"enode://ce9c87cfe089f6811d26c96913fa3ec10b938d9017fc6246684c74a33679ee34ceca9447180fb509e37bf2b706c2877a82085d34bfd83b5b520ee1288b0fc32f@35.198.109.49:30303",
	"enode://713657eb6a53feadcbc47e634ad557326a51eb6818a3e19a00a8111492f50a666ccbf2f5d334d247ecf941e68d242ef5c3b812b63c44d381ef11f79c2cdb45c7@34.141.15.100:30303",

	// Canada
	"enode://d071fa740e063ce1bb9cdc2b7937baeff6dc4000f91588d730a731c38a6ff0d4015814812c160fab8695e46f74b9b618735368ea2f16db4d785f16d29b3fb7b0@35.203.2.210:30303",
	"enode://ffc452fe451a2e5f89fe634744aea334d92dcd30d881b76209d2db7dbf4b7ee047e7c69a5bb1633764d987a7441d9c4bc57ccdbfd6442a2f860bf953bc89a9b9@34.152.50.224:30303",
	"enode://da94328302a1d1422209d1916744e90b6095a48b2340dcec39b22002c098bb4d58a880dab98eb26edf03fa4705d1b62f99a8c5c14e6666e4726b6d3066d8a4d7@34.95.61.106:30303",
	"enode://19c7671a4844699b481e81a5bcfe7bafc7fefa953c16ebbe1951b1046371e73839e9058de6b7d3c934318fe7e7233dde3621c1c1018eb8b294ea3d4516147150@35.203.82.137:30303",
}

// BepoliaBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Bepolia testnet.
var BepoliaBootnodes = []string{
	"enode://879752877fe08e483eba228b7544ff2de4a90e8ddaa59f122dfbc5c3689ea3030a0bfe4532d6a411a936dadce84e7a89086dca39dc39b9119603d87469010d9e@34.152.29.103:30303",
	"enode://11f28864a15162c66b4b23d20b91fe08fda1d541df778d8db33be72558e80489a785d3fcc870926d10b7a6b848c7c6319bc215481319e527ce54ac139c2f44dc@34.118.159.247:30303",
	"enode://47f41b9ab5a45e880a78330d2ae3f95a61f5cb41f203bbc9c9ff0e37778fc6c7fd46a6ee103e65ac36df8c024075a33f535a03cd8d800800c27fa2699fa0182b@34.47.28.251:30303",
	"enode://fe6d2429b582de7daf387c6e5436f05d9185965267b72b8b6b4924125b50afdad765a820213d73d2afbfe64a721160a1a94750b4874ed97ccbe97c51443d1c42@34.95.21.165:30303",
	"enode://1273d68cb5a884630aff8c35a30f643ad739461a5f7c8d2d691dc65547d5ad023829e1b6bf128edddd032047cb5c13e9dcd0f93a5de4aa5b958cef5970541863@34.159.207.112:30303",
	"enode://fac9c2a0f719ec0ebc9021e88ad32eb7c05f4f351993dab6e4c86ae3ef5a47cade21749281263221c2b5583437893f957ac282858fd8a00f75f9a495eb53ae4e@34.141.54.42:30303",
	"enode://5c0d582c19ea9f19928cfd6b7e156372d051b5720f67a444c38b671b8119bb097abfd1e4b868a389ab4a65bb9b405ef837df0ae195c0f32a33c61c39ca54e8ab@34.107.71.151:30303",
	"enode://59aec227e87f4cd7c0a24c6cb0f870ef77abcc0c6d640f5a536c597a206b3505f7963f6459c265952b7bed3c3f260edf8a1412bbdb05f0e59d6bd612dc4bf077@34.141.48.88:30303",
	"enode://281e3927839de398cb571279b324b6251d92c8de79263648f24dd4e451499a7fb336354bd4b7dc616e439f804cdd518fe49789f20b060c6426020de2c1cb285a@35.240.139.2:30303",
	"enode://ae3f2e3f2caffc82aa0a5f5c6036def3d0be046b9245651b61567b4743c3c4937c08cb075d4ac543b163045942028384f3ef93bb25f7d500b644a2aab78f6b46@34.87.140.47:30303",
	"enode://59e79f22bbf1645c60666ba6c17c54845080473df80711807fb3a4fbefc26278f78c43fa44ea6e52db38c57f1fccf6caa2de7589ca6cce8e161c3120e7a8d0d8@35.198.214.183:30303",
	"enode://91d6e1878f5ab434a36625ac9017b792179324d42f4650a8dcc38774578d7ffb68882ec56861155e738eb62a7365b8463bd26886523f60edcfbf6137b43b9b65@35.240.234.137:30303",
	"enode://f55dc8bac635b9c3b9fad688e3200c6b0744e2930d8d8a8e91d1effc0cd91b0547da3eac6d8d1431f0dd130ccc20f3c856fc90e32002bfc1e27d3286eef8c570@34.47.125.153:30303",
	"enode://13da5e4ff1ab3481966f8d83309d8bd898ef5fa56355da57e683bdcc26b4c205be40341c87236934610d0c0bdddf11f6cef495966bba0092465839b8942bc54c@34.47.93.104:30303",
	"enode://c2d43cba3380df975e78befc7294fe07a042eab1c2b743bdf436274b884bcdd0ef218821ff24732ff2106bf796964ada13dbeb42e27756f2491710be402fce59@34.64.123.223:30303",
	"enode://f5b93e79932028a30bacb1b1cd7e7d42b30fe69640c0081bb51e29483bfe715c698dfcf50a309f208743c0ef07884afa39d0c96c6e56a8e58ce7ff1181fdffa0@34.22.89.170:30303",
}

var V5Bootnodes = []string{
	// Teku team's bootnode
	"enr:-KG4QMOEswP62yzDjSwWS4YEjtTZ5PO6r65CPqYBkgTTkrpaedQ8uEUo1uMALtJIvb2w_WWEVmg5yt1UAuK1ftxUU7QDhGV0aDKQu6TalgMAAAD__________4JpZIJ2NIJpcIQEnfA2iXNlY3AyNTZrMaEDfol8oLr6XJ7FsdAYE7lpJhKMls4G_v6qQOGKJUWGb_uDdGNwgiMog3VkcIIjKA", // # 4.157.240.54 | azure-us-east-virginia
	"enr:-KG4QF4B5WrlFcRhUU6dZETwY5ZzAXnA0vGC__L1Kdw602nDZwXSTs5RFXFIFUnbQJmhNGVU6OIX7KVrCSTODsz1tK4DhGV0aDKQu6TalgMAAAD__________4JpZIJ2NIJpcIQExNYEiXNlY3AyNTZrMaECQmM9vp7KhaXhI-nqL_R0ovULLCFSFTa9CPPSdb1zPX6DdGNwgiMog3VkcIIjKA", // 4.196.214.4  | azure-au-east-sydney
	// Prylab team's bootnodes
	"enr:-Ku4QImhMc1z8yCiNJ1TyUxdcfNucje3BGwEHzodEZUan8PherEo4sF7pPHPSIB1NNuSg5fZy7qFsjmUKs2ea1Whi0EBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQOVphkDqal4QzPMksc5wnpuC3gvSC8AfbFOnZY_On34wIN1ZHCCIyg", // 18.223.219.100 | aws-us-east-2-ohio
	"enr:-Ku4QP2xDnEtUXIjzJ_DhlCRN9SN99RYQPJL92TMlSv7U5C1YnYLjwOQHgZIUXw6c-BvRg2Yc2QsZxxoS_pPRVe0yK8Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQMeFF5GrS7UZpAH2Ly84aLK-TyvH-dRo0JM1i8yygH50YN1ZHCCJxA", // 18.223.219.100 | aws-us-east-2-ohio
	"enr:-Ku4QPp9z1W4tAO8Ber_NQierYaOStqhDqQdOPY3bB3jDgkjcbk6YrEnVYIiCBbTxuar3CzS528d2iE7TdJsrL-dEKoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQMw5fqqkw2hHC4F5HZZDPsNmPdB1Gi8JPQK7pRc9XHh-oN1ZHCCKvg", // 18.223.219.100 | aws-us-east-2-ohio
	// Lighthouse team's bootnodes
	"enr:-Le4QPUXJS2BTORXxyx2Ia-9ae4YqA_JWX3ssj4E_J-3z1A-HmFGrU8BpvpqhNabayXeOZ2Nq_sbeDgtzMJpLLnXFgAChGV0aDKQtTA_KgEAAAAAIgEAAAAAAIJpZIJ2NIJpcISsaa0Zg2lwNpAkAIkHAAAAAPA8kv_-awoTiXNlY3AyNTZrMaEDHAD2JKYevx89W0CcFJFiskdcEzkH_Wdv9iW42qLK79ODdWRwgiMohHVkcDaCI4I", // 172.105.173.25 | linode-au-sydney
	"enr:-Le4QLHZDSvkLfqgEo8IWGG96h6mxwe_PsggC20CL3neLBjfXLGAQFOPSltZ7oP6ol54OvaNqO02Rnvb8YmDR274uq8ChGV0aDKQtTA_KgEAAAAAIgEAAAAAAIJpZIJ2NIJpcISLosQxg2lwNpAqAX4AAAAAAPA8kv_-ax65iXNlY3AyNTZrMaEDBJj7_dLFACaxBfaI8KZTh_SSJUjhyAyfshimvSqo22WDdWRwgiMohHVkcDaCI4I", // 139.162.196.49 | linode-uk-london
	"enr:-Le4QH6LQrusDbAHPjU_HcKOuMeXfdEB5NJyXgHWFadfHgiySqeDyusQMvfphdYWOzuSZO9Uq2AMRJR5O4ip7OvVma8BhGV0aDKQtTA_KgEAAAAAIgEAAAAAAIJpZIJ2NIJpcISLY9ncg2lwNpAkAh8AgQIBAAAAAAAAAAmXiXNlY3AyNTZrMaECDYCZTZEksF-kmgPholqgVt8IXr-8L7Nu7YrZ7HUpgxmDdWRwgiMohHVkcDaCI4I", // 139.99.217.220 | ovh-au-sydney
	"enr:-Le4QIqLuWybHNONr933Lk0dcMmAB5WgvGKRyDihy1wHDIVlNuuztX62W51voT4I8qD34GcTEOTmag1bcdZ_8aaT4NUBhGV0aDKQtTA_KgEAAAAAIgEAAAAAAIJpZIJ2NIJpcISLY04ng2lwNpAkAh8AgAIBAAAAAAAAAA-fiXNlY3AyNTZrMaEDscnRV6n1m-D9ID5UsURk0jsoKNXt1TIrj8uKOGW6iluDdWRwgiMohHVkcDaCI4I", // 139.99.78.39 | ovh-singapore
	// EF bootnodes
	"enr:-Ku4QHqVeJ8PPICcWk1vSn_XcSkjOkNiTg6Fmii5j6vUQgvzMc9L1goFnLKgXqBJspJjIsB91LTOleFmyWWrFVATGngBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhAMRHkWJc2VjcDI1NmsxoQKLVXFOhp2uX6jeT0DvvDpPcU8FWMjQdR4wMuORMhpX24N1ZHCCIyg", // 3.17.30.69 | aws-us-east-2-ohio
	"enr:-Ku4QG-2_Md3sZIAUebGYT6g0SMskIml77l6yR-M_JXc-UdNHCmHQeOiMLbylPejyJsdAPsTHJyjJB2sYGDLe0dn8uYBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhBLY-NyJc2VjcDI1NmsxoQORcM6e19T1T9gi7jxEZjk_sjVLGFscUNqAY9obgZaxbIN1ZHCCIyg", // 18.216.248.220 | aws-us-east-2-ohio
	"enr:-Ku4QPn5eVhcoF1opaFEvg1b6JNFD2rqVkHQ8HApOKK61OIcIXD127bKWgAtbwI7pnxx6cDyk_nI88TrZKQaGMZj0q0Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhDayLMaJc2VjcDI1NmsxoQK2sBOLGcUb4AwuYzFuAVCaNHA-dy24UuEKkeFNgCVCsIN1ZHCCIyg", // 54.178.44.198 | aws-ap-northeast-1-tokyo
	"enr:-Ku4QEWzdnVtXc2Q0ZVigfCGggOVB2Vc1ZCPEc6j21NIFLODSJbvNaef1g4PxhPwl_3kax86YPheFUSLXPRs98vvYsoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhDZBrP2Jc2VjcDI1NmsxoQM6jr8Rb1ktLEsVcKAPa08wCsKUmvoQ8khiOl_SLozf9IN1ZHCCIyg", // 54.65.172.253 | aws-ap-northeast-1-tokyo
	// Nimbus team's bootnodes
	"enr:-LK4QA8FfhaAjlb_BXsXxSfiysR7R52Nhi9JBt4F8SPssu8hdE1BXQQEtVDC3qStCW60LSO7hEsVHv5zm8_6Vnjhcn0Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhAN4aBKJc2VjcDI1NmsxoQJerDhsJ-KxZ8sHySMOCmTO6sHM3iCFQ6VMvLTe948MyYN0Y3CCI4yDdWRwgiOM", // 3.120.104.18 | aws-eu-central-1-frankfurt
	"enr:-LK4QKWrXTpV9T78hNG6s8AM6IO4XH9kFT91uZtFg1GcsJ6dKovDOr1jtAAFPnS2lvNltkOGA9k29BUN7lFh_sjuc9QBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhANAdd-Jc2VjcDI1NmsxoQLQa6ai7y9PMN5hpLe5HmiJSlYzMuzP7ZhwRiwHvqNXdoN0Y3CCI4yDdWRwgiOM", // 3.64.117.223 | aws-eu-central-1-frankfurt}
}

const dnsPrefix = "enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@"

// KnownDNSNetwork returns the address of a public DNS-based node list for the given
// genesis hash and protocol. See https://github.com/ethereum/discv4-dns-lists for more
// information.
//
// TODO: add support for bepolia and berachain mainnet.
func KnownDNSNetwork(genesis common.Hash, protocol string) string {
	var net string
	switch genesis {
	case MainnetGenesisHash:
		net = "mainnet"
	case SepoliaGenesisHash:
		net = "sepolia"
	case HoleskyGenesisHash:
		net = "holesky"
	case HoodiGenesisHash:
		net = "hoodi"
	default:
		return ""
	}
	return dnsPrefix + protocol + "." + net + ".ethdisco.net"
}
