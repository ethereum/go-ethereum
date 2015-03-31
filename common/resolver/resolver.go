package resolver

import (
	"fmt"
	"net/url"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/xeth"
)

/*
Resolver implements the Ethereum DNS mapping
NameReg : Domain Name (or Code hash of Contract) -> Content Hash
UrlHint : Content Hash -> Url Hint
*/
const (
	urlHintContractAddress = "urlhint"
	nameRegContractAddress = "nameReg"
)

type Resolver struct {
	xeth *xeth.XEth
}

func (self *Resolver) NameToContentHash(name string) (hash common.Hash, err error) {
	// look up in nameReg
	copy(hash[:], []byte(name)[:32])
	return
}

func (self *Resolver) ContentHashToUrl(hash common.Hash) (uri *url.URL, err error) {
	// look up in nameReg
	rawurl := fmt.Sprintf("bzz://%x/my/path/mycontract.sud", hash[:])
	// mime type?
	return url.Parse(rawurl)
}

func (self *Resolver) NameToUrl(name string) (uri *url.URL, err error) {
	// look up in urlHint
	hash, err := self.NameToContentHash(name)
	if err != nil {
		return
	}
	return self.ContentHashToUrl(hash)
}
