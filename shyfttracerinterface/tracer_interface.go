package shyfttracerinterface

type IShyftTracer interface {
	MyTraceTransaction(hash string) (interface{})
}
