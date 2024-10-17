package history

import (
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type API struct {
	*discover.PortalProtocolAPI
}

func (p *API) HistoryRoutingTableInfo() *discover.RoutingTableInfo {
	return p.RoutingTableInfo()
}

func (p *API) HistoryAddEnr(enr string) (bool, error) {
	return p.AddEnr(enr)
}

func (p *API) HistoryGetEnr(nodeId string) (string, error) {
	return p.GetEnr(nodeId)
}

func (p *API) HistoryDeleteEnr(nodeId string) (bool, error) {
	return p.DeleteEnr(nodeId)
}

func (p *API) HistoryLookupEnr(nodeId string) (string, error) {
	return p.LookupEnr(nodeId)
}

func (p *API) HistoryPing(enr string) (*discover.PortalPongResp, error) {
	return p.Ping(enr)
}

func (p *API) HistoryFindNodes(enr string, distances []uint) ([]string, error) {
	return p.FindNodes(enr, distances)
}

func (p *API) HistoryFindContent(enr string, contentKey string) (interface{}, error) {
	return p.FindContent(enr, contentKey)
}

func (p *API) HistoryOffer(enr string, contentItems [][2]string) (string, error) {
	return p.Offer(enr, contentItems)
}

func (p *API) HistoryRecursiveFindNodes(nodeId string) ([]string, error) {
	return p.RecursiveFindNodes(nodeId)
}

func (p *API) HistoryGetContent(contentKeyHex string) (*discover.ContentInfo, error) {
	return p.RecursiveFindContent(contentKeyHex)
}

func (p *API) HistoryLocalContent(contentKeyHex string) (string, error) {
	return p.LocalContent(contentKeyHex)
}

func (p *API) HistoryStore(contentKeyHex string, contextHex string) (bool, error) {
	return p.Store(contentKeyHex, contextHex)
}

func (p *API) HistoryGossip(contentKeyHex, contentHex string) (int, error) {
	return p.Gossip(contentKeyHex, contentHex)
}

func (p *API) HistoryTraceGetContent(contentKeyHex string) (*discover.TraceContentResult, error) {
	return p.TraceRecursiveFindContent(contentKeyHex)
}

func NewHistoryNetworkAPI(historyAPI *discover.PortalProtocolAPI) *API {
	return &API{
		historyAPI,
	}
}
