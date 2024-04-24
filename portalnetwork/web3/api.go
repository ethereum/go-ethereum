package web3

import "runtime"

type API struct{}

func (p *API) ClientVersion() string {
	// TODO add version
	name := "Shisui"
	name += "/" + runtime.GOOS + "-" + runtime.GOARCH
	name += "/" + runtime.Version()
	return name
}
