package singlepoll

import "context"

var backgroundctx, backgroundcancel = context.WithCancel(context.Background())

func CloseBackgroundProcesses() {
	backgroundcancel()
}
