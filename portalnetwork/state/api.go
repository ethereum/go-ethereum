package state

import (
	"github.com/ethereum/go-ethereum/portalnetwork/portalwire"
)

type API struct {
	*portalwire.PortalProtocolAPI
}

func (p *API) StateRoutingTableInfo() *portalwire.RoutingTableInfo {
	return p.RoutingTableInfo()
}

func (p *API) StateAddEnr(enr string) (bool, error) {
	return p.AddEnr(enr)
}

func (p *API) StateGetEnr(nodeId string) (string, error) {
	return p.GetEnr(nodeId)
}

func (p *API) StateDeleteEnr(nodeId string) (bool, error) {
	return p.DeleteEnr(nodeId)
}

func (p *API) StateLookupEnr(nodeId string) (string, error) {
	return p.LookupEnr(nodeId)
}

func (p *API) StatePing(enr string) (*portalwire.PortalPongResp, error) {
	return p.Ping(enr)
}

func (p *API) StateFindNodes(enr string, distances []uint) ([]string, error) {
	return p.FindNodes(enr, distances)
}

func (p *API) StateFindContent(enr string, contentKey string) (interface{}, error) {
	return p.FindContent(enr, contentKey)
}

func (p *API) StateOffer(enr string, contentItems [][2]string) (string, error) {
	return p.Offer(enr, contentItems)
}

func (p *API) StateRecursiveFindNodes(nodeId string) ([]string, error) {
	return p.RecursiveFindNodes(nodeId)
}

func (p *API) StateGetContent(contentKeyHex string) (*portalwire.ContentInfo, error) {
	return p.RecursiveFindContent(contentKeyHex)
}

func (p *API) StateLocalContent(contentKeyHex string) (string, error) {
	return p.LocalContent(contentKeyHex)
}

func (p *API) StateStore(contentKeyHex string, contextHex string) (bool, error) {
	return p.Store(contentKeyHex, contextHex)
}

func (p *API) StateGossip(contentKeyHex, contentHex string) (int, error) {
	return p.Gossip(contentKeyHex, contentHex)
}

func (p *API) StateTraceGetContent(contentKeyHex string) (*portalwire.TraceContentResult, error) {
	return p.TraceRecursiveFindContent(contentKeyHex)
}

func NewStateNetworkAPI(portalProtocolAPI *portalwire.PortalProtocolAPI) *API {
	return &API{
		portalProtocolAPI,
	}
}
