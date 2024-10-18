package discover

import "github.com/ethereum/go-ethereum/metrics"

type portalMetrics struct {
	messagesReceivedAccept      metrics.Meter
	messagesReceivedNodes       metrics.Meter
	messagesReceivedFindNodes   metrics.Meter
	messagesReceivedFindContent metrics.Meter
	messagesReceivedContent     metrics.Meter
	messagesReceivedOffer       metrics.Meter
	messagesReceivedPing        metrics.Meter
	messagesReceivedPong        metrics.Meter

	messagesSentAccept      metrics.Meter
	messagesSentNodes       metrics.Meter
	messagesSentFindNodes   metrics.Meter
	messagesSentFindContent metrics.Meter
	messagesSentContent     metrics.Meter
	messagesSentOffer       metrics.Meter
	messagesSentPing        metrics.Meter
	messagesSentPong        metrics.Meter

	utpInFailConn     metrics.Counter
	utpInFailTx       metrics.Counter
	utpInFailShutdown metrics.Counter
	utpInSuccess      metrics.Counter

	utpOutFailConn     metrics.Counter
	utpOutFailTx       metrics.Counter
	utpOutFailShutdown metrics.Counter
	utpOutSuccess      metrics.Counter

	contentValidated   metrics.Counter
	contentInvalidated metrics.Counter
}

func newPortalMetrics(protocolName string) *portalMetrics {
	return &portalMetrics{
		messagesReceivedAccept:      metrics.NewRegisteredMeter("portal/"+protocolName+"/received/accept", nil),
		messagesReceivedNodes:       metrics.NewRegisteredMeter("portal/"+protocolName+"/received/nodes", nil),
		messagesReceivedFindNodes:   metrics.NewRegisteredMeter("portal/"+protocolName+"/received/find_nodes", nil),
		messagesReceivedFindContent: metrics.NewRegisteredMeter("portal/"+protocolName+"/received/find_content", nil),
		messagesReceivedContent:     metrics.NewRegisteredMeter("portal/"+protocolName+"/received/content", nil),
		messagesReceivedOffer:       metrics.NewRegisteredMeter("portal/"+protocolName+"/received/offer", nil),
		messagesReceivedPing:        metrics.NewRegisteredMeter("portal/"+protocolName+"/received/ping", nil),
		messagesReceivedPong:        metrics.NewRegisteredMeter("portal/"+protocolName+"/received/pong", nil),
		messagesSentAccept:          metrics.NewRegisteredMeter("portal/"+protocolName+"/sent/accept", nil),
		messagesSentNodes:           metrics.NewRegisteredMeter("portal/"+protocolName+"/sent/nodes", nil),
		messagesSentFindNodes:       metrics.NewRegisteredMeter("portal/"+protocolName+"/sent/find_nodes", nil),
		messagesSentFindContent:     metrics.NewRegisteredMeter("portal/"+protocolName+"/sent/find_content", nil),
		messagesSentContent:         metrics.NewRegisteredMeter("portal/"+protocolName+"/sent/content", nil),
		messagesSentOffer:           metrics.NewRegisteredMeter("portal/"+protocolName+"/sent/offer", nil),
		messagesSentPing:            metrics.NewRegisteredMeter("portal/"+protocolName+"/sent/ping", nil),
		messagesSentPong:            metrics.NewRegisteredMeter("portal/"+protocolName+"/sent/pong", nil),
		utpInFailConn:               metrics.NewRegisteredCounter("portal/"+protocolName+"/utp/inbound/fail_conn", nil),
		utpInFailTx:                 metrics.NewRegisteredCounter("portal/"+protocolName+"/utp/inbound/fail_tx", nil),
		utpInFailShutdown:           metrics.NewRegisteredCounter("portal/"+protocolName+"/utp/inbound/fail_shutdown", nil),
		utpInSuccess:                metrics.NewRegisteredCounter("portal/"+protocolName+"/utp/inbound/success", nil),
		utpOutFailConn:              metrics.NewRegisteredCounter("portal/"+protocolName+"/utp/outbound/fail_conn", nil),
		utpOutFailTx:                metrics.NewRegisteredCounter("portal/"+protocolName+"/utp/outbound/fail_tx", nil),
		utpOutFailShutdown:          metrics.NewRegisteredCounter("portal/"+protocolName+"/utp/outbound/fail_shutdown", nil),
		utpOutSuccess:               metrics.NewRegisteredCounter("portal/"+protocolName+"/utp/outbound/success", nil),
		contentValidated:            metrics.NewRegisteredCounter("portal/"+protocolName+"/content/validated", nil),
		contentInvalidated:          metrics.NewRegisteredCounter("portal/"+protocolName+"/content/invalidated", nil),
	}
}
