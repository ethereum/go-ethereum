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
	"enr:-KG4QCF1Mj32xpKHjinNb6ocCtMZG6IR_tyF5dkio5Hkek7zVbT6MM5eJwhjJFdiksQl51T33IRgryE0XLXiy1QOqsUBgmlkgnY0gmlwhNRj2kKDaXA2kCoAHKALAA0CAAAAAAAAAF6Jc2VjcDI1NmsxoQLKlnQYuhZRBTA8-7cz37kr_KuA1lAJ1eXxWMjp5fLJB4N1ZHCCTreEdWRwNoJOtw", // nodeops-bootnode-dcl1-01
	"enr:-KG4QMHihBfLI_tpXLWmfTi28HItJa4wDADDAaVjkfSELAb0BajqVWXM2GEMds5J63SCkpDRlqct5E8ftfQi4U93pigBgmlkgnY0gmlwhIHUpj2DaXA2kCYEqIAABAHQAAAAA2GdUACJc2VjcDI1NmsxoQLeQmW8OMuoUIoUImNW-r9IC4jaiAwZ2knfV4BqWyQvKoN1ZHCCdl-EdWRwNoJ2Xw", // nodeops-bootnode-sfo3-01
	"enr:-KG4QKsHcKQvIX8UPbJgw2cIEP_9EislF1srNGdYom4dcSsobt-92iEDNkHTt9dcdKXHaDMVbCM8QQ4V1RP8HcExLDsBgmlkgnY0gmlwhJB-_BiDaXA2kCQAYYABAADQAAAAAYEgYAGJc2VjcDI1NmsxoQILdc6UClD5s_N8-hYhIGQtFqRYR3Qxf9qgLQ7sj4qvEYN1ZHCCdl-EdWRwNoJ2Xw", // nodeops-bootnode-blr1-01
	"enr:-KG4QFplptv1jVEhFeNNxRN7qe5p9rAGGlMeNnmGvUJV23W4Kr8hmiSzdWjZZHFqwi7nOuQrM32VO6uHBORXOdLG98gBgmlkgnY0gmlwhLKc14yDaXA2kCoBBP8A9DxKAAAAAAAAAAGJc2VjcDI1NmsxoQMBRycp4yiHbGzOqZPPMmuKGslp1NpOrFf_predLYM4-oN1ZHCCdl-EdWRwNoJ2Xw", // nodeops-bootnode-ash-01
	"enr:-KG4QGY74qTrst5NbQDAMIF8B7jEC8CMBSo6C5Ljea5I4PxYV1VSYdsGUqqIBlOYRBfwhl6YpEyvvvZOqNCa4dvmSXgBgmlkgnY0gmlwhAXfXlGDaXA2kCoBBP8C8BytAAAAAAAAAAGJc2VjcDI1NmsxoQIMlJp7yNcblauYicEeSBpHQzFAk9Wg3lhx3ie9gpTohIN1ZHCCdl-EdWRwNoJ2Xw", // nodeops-bootnode-sin-01
}

// HoodiBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Hoodi test network.
var HoodiBootnodes = []string{
	// EF DevOps
	"enr:-KG4QHqPRT-PQyT1l2-DzAJ4WiSGWDbLvzqV8umrSRmBVvzfaYAIjswgIY3S4iZFBtOGGdFeUXh5uaFXI4FdN4ib-S0BgmlkgnY0gmlwhNRj2kKDaXA2kCoAHKALAA0CAAAAAAAAAF6Jc2VjcDI1NmsxoQNwurkRdfm7zrv78VVkT0bAPtRO9JJ9gW42Abpf0y96JIN1ZHCCTrmEdWRwNoJOuQ", // nodeops-bootnode-dcl1-01
	"enr:-KG4QO3rOpovULG1tqfd5prt30FBF_0_dbh6z13W1CS94yZvObO3BDatF6v_5p0Fol4ayvf32rtLy9IoeIrllRNO6mEBgmlkgnY0gmlwhIHUpj2DaXA2kCYEqIAABAHQAAAAA2GdUACJc2VjcDI1NmsxoQKv1QQH20VixQSfOvSJdDoktzLLpq8hleYiBb5ZrmRaoIN1ZHCCdyeEdWRwNoJ3Jw", // nodeops-bootnode-sfo3-01
	"enr:-KG4QK3lmOaDQ5f-OLtwBQDwZZwORygbjDKbSWR79YFuQBQMbdStUfUKBucPPB8BW2mIZPcAHmsKan6HK9_CLv1ntfsBgmlkgnY0gmlwhJB-_BiDaXA2kCQAYYABAADQAAAAAYEgYAGJc2VjcDI1NmsxoQL3h68BwVT7D-gnmNXHLKgePyp4l5ikdqbJPaesuz9YAIN1ZHCCdyeEdWRwNoJ3Jw", // nodeops-bootnode-blr1-01
	"enr:-KG4QC1VbcxVmJVOBrQFyaTshr7kGgI8_IgGhkXMlPGQN6ijWpc72mLFR9qx6islo-bcG5gcfJDBA85Pjjk3zXdhsckBgmlkgnY0gmlwhLKc14yDaXA2kCoBBP8A9DxKAAAAAAAAAAGJc2VjcDI1NmsxoQJxxADpm6ox6R2Op72lhQ7xQ7nalLB0E9vUwchMS1_EdIN1ZHCCdyeEdWRwNoJ3Jw", // nodeops-bootnode-ash-01
	"enr:-KG4QACXfYMzVZrG5YGEM6JoAyALZrmSG_0rtTtRQK3Z7q3TUYgb1gqEbyXfA07HpqJq9i4ZaG1EmRwgZbwNFIa9M8gBgmlkgnY0gmlwhAXfXlGDaXA2kCoBBP8C8BytAAAAAAAAAAGJc2VjcDI1NmsxoQOBqr1yI_NF5xO381XMdmgLqOw6ewmPEAZVpGfkBJETWIN1ZHCCdyeEdWRwNoJ3Jw", // nodeops-bootnode-sin-01
}

// SepoliaBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Sepolia test network.
var SepoliaBootnodes = []string{
	// EF DevOps
	"enr:-KG4QHn1Os72NN5Z44Ehf9CHPmN490GBphTFirfk33pFuWnkY9UNrQMfHkuCw6a-ykeRhGKrPL5ppMXiZQchgPrCDJQBgmlkgnY0gmlwhNRj2kKDaXA2kCoAHKALAA0CAAAAAAAAAF6Jc2VjcDI1NmsxoQJK_ye9jx9mela-ME_Kt5e017Ywv3hYH_5rwNhIUrtzNoN1ZHCCTriEdWRwNoJOuA", // nodeops-bootnode-dcl1-01
	"enr:-KG4QCpiRltiyX7LgozCkQ8XBrcDzejBXbE7YRUQFmIEgv46SC8B4CJLcOO3O2bH-_u-B242GTzO_hQPhv3rpyvAHXMBgmlkgnY0gmlwhIHUpj2DaXA2kCYEqIAABAHQAAAAA2GdUACJc2VjcDI1NmsxoQNmVWXve5c0uvsn_agjTKQ8pR6gl9DynZyTQNruBDfJ4YN1ZHCCdsOEdWRwNoJ2ww", // nodeops-bootnode-sfo3-01
	"enr:-KG4QJUWvc8qvCUr7x-qCFWee6HPLgmRUfVn-M_yqXD-sjSoAalKCV3NgUhmlqozjTJIj_nPr7J_uT8WJwbWiIx3Cv4BgmlkgnY0gmlwhJB-_BiDaXA2kCQAYYABAADQAAAAAYEgYAGJc2VjcDI1NmsxoQKOQetrA-97TELUzuGegVD1_cHKKNnjOmJ9CYdeSTor7YN1ZHCCdsOEdWRwNoJ2ww", // nodeops-bootnode-blr1-01
	"enr:-KG4QIBsIiST0JBBvgSNxvHQOeH5PkRgWeufahVXYQQWj7GLYPFDsviuQ6qU-3Xx4qnARaZiI-qqVRyw_2qqmOKRY1UBgmlkgnY0gmlwhLKc14yDaXA2kCoBBP8A9DxKAAAAAAAAAAGJc2VjcDI1NmsxoQKx4n3wy0KtwnuZCHmlxMmc4b0VvfC4Ka_f3sUPvjUq1oN1ZHCCdsOEdWRwNoJ2ww", // nodeops-bootnode-ash-01
	"enr:-KG4QPM5C6wuTxu5yk4jkI9Mm64iSUKMPNt8y8kmoFzOlYrnYV4SBeZK7UiS4YWrkX4y2FD9_eH7xiPGvLQ-nXoXIcYBgmlkgnY0gmlwhAXfXlGDaXA2kCoBBP8C8BytAAAAAAAAAAGJc2VjcDI1NmsxoQIC3FMD8Si9DIBVofsSbJkpvXTUAfyi8q66XXQgjRhUsIN1ZHCCdsOEdWRwNoJ2ww", // nodeops-bootnode-sin-01
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
	"enr:-KG4QCF1Mj32xpKHjinNb6ocCtMZG6IR_tyF5dkio5Hkek7zVbT6MM5eJwhjJFdiksQl51T33IRgryE0XLXiy1QOqsUBgmlkgnY0gmlwhNRj2kKDaXA2kCoAHKALAA0CAAAAAAAAAF6Jc2VjcDI1NmsxoQLKlnQYuhZRBTA8-7cz37kr_KuA1lAJ1eXxWMjp5fLJB4N1ZHCCTreEdWRwNoJOtw", // 212.99.218.66 | colo-dcl1
	"enr:-KG4QMHihBfLI_tpXLWmfTi28HItJa4wDADDAaVjkfSELAb0BajqVWXM2GEMds5J63SCkpDRlqct5E8ftfQi4U93pigBgmlkgnY0gmlwhIHUpj2DaXA2kCYEqIAABAHQAAAAA2GdUACJc2VjcDI1NmsxoQLeQmW8OMuoUIoUImNW-r9IC4jaiAwZ2knfV4BqWyQvKoN1ZHCCdl-EdWRwNoJ2Xw", // 129.212.166.61 | digitalocean-sfo3
	"enr:-KG4QKsHcKQvIX8UPbJgw2cIEP_9EislF1srNGdYom4dcSsobt-92iEDNkHTt9dcdKXHaDMVbCM8QQ4V1RP8HcExLDsBgmlkgnY0gmlwhJB-_BiDaXA2kCQAYYABAADQAAAAAYEgYAGJc2VjcDI1NmsxoQILdc6UClD5s_N8-hYhIGQtFqRYR3Qxf9qgLQ7sj4qvEYN1ZHCCdl-EdWRwNoJ2Xw", // 144.126.252.24 | digitalocean-blr1
	"enr:-KG4QFplptv1jVEhFeNNxRN7qe5p9rAGGlMeNnmGvUJV23W4Kr8hmiSzdWjZZHFqwi7nOuQrM32VO6uHBORXOdLG98gBgmlkgnY0gmlwhLKc14yDaXA2kCoBBP8A9DxKAAAAAAAAAAGJc2VjcDI1NmsxoQMBRycp4yiHbGzOqZPPMmuKGslp1NpOrFf_predLYM4-oN1ZHCCdl-EdWRwNoJ2Xw", // 178.156.215.140 | hetzner-ash
	"enr:-KG4QGY74qTrst5NbQDAMIF8B7jEC8CMBSo6C5Ljea5I4PxYV1VSYdsGUqqIBlOYRBfwhl6YpEyvvvZOqNCa4dvmSXgBgmlkgnY0gmlwhAXfXlGDaXA2kCoBBP8C8BytAAAAAAAAAAGJc2VjcDI1NmsxoQIMlJp7yNcblauYicEeSBpHQzFAk9Wg3lhx3ie9gpTohIN1ZHCCdl-EdWRwNoJ2Xw", // 5.223.94.81 | hetzner-sin
	// Nimbus team's bootnodes
	"enr:-LK4QA8FfhaAjlb_BXsXxSfiysR7R52Nhi9JBt4F8SPssu8hdE1BXQQEtVDC3qStCW60LSO7hEsVHv5zm8_6Vnjhcn0Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhAN4aBKJc2VjcDI1NmsxoQJerDhsJ-KxZ8sHySMOCmTO6sHM3iCFQ6VMvLTe948MyYN0Y3CCI4yDdWRwgiOM", // 3.120.104.18 | aws-eu-central-1-frankfurt
	"enr:-LK4QKWrXTpV9T78hNG6s8AM6IO4XH9kFT91uZtFg1GcsJ6dKovDOr1jtAAFPnS2lvNltkOGA9k29BUN7lFh_sjuc9QBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhANAdd-Jc2VjcDI1NmsxoQLQa6ai7y9PMN5hpLe5HmiJSlYzMuzP7ZhwRiwHvqNXdoN0Y3CCI4yDdWRwgiOM", // 3.64.117.223 | aws-eu-central-1-frankfurt}
}

const dnsPrefix = "enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@"

// KnownDNSNetwork returns the address of a public DNS-based node list for the given
// genesis hash and protocol. See https://github.com/ethereum/discv4-dns-lists for more
// information.
func KnownDNSNetwork(genesis common.Hash, protocol string) string {
	var net string
	switch genesis {
	case MainnetGenesisHash:
		net = "mainnet"
	case SepoliaGenesisHash:
		net = "sepolia"
	case HoodiGenesisHash:
		net = "hoodi"
	default:
		return ""
	}
	return dnsPrefix + protocol + "." + net + ".ethdisco.net"
}
