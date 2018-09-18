// +build withserver

package stream

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/storage"
	//	"github.com/ethereum/go-ethereum/swarm/tracing"
)

func init() {
	/*
		var flagSet *flag.FlagSet
				tracing.Enabled = true
				tracing.StandaloneSetup()
					fakeApp := cli.NewApp()
					flags := []cli.Flag{
						tracing.TracingEndpointFlag,
						tracing.TracingSvcFlag,
					}
					fakeApp.Flags = append(fakeApp.Flags, flags...)
					fakeApp.Before = func(ctx *cli.Context) error {
						tracing.Setup(ctx)
						return nil
					}
					fakeApp.Run([]string{"-tracing.endpoint", tracing.TracingEndpointFlag.Value, "-tracing.svc", tracing.TracingSvcFlag.Value})

					//flagSet = flag.NewFlagSet("traceFlags", 0)
					//tracing.Setup(cli.NewContext(fakeApp, flagSet, nil))
	*/
}

func setupSim(serviceMap map[string]simulation.ServiceFunc) (int, int, *simulation.Simulation) {
	nodeCount := *nodes
	chunkCount := *chunks

	if nodeCount == 0 || chunkCount == 0 {
		nodeCount = 32
		chunkCount = 1
	}

	sim := simulation.New(serviceMap).WithServer(":8888")
	return nodeCount, chunkCount, sim
}

func watchSim(sim *simulation.Simulation) (context.Context, context.CancelFunc) {
	ctx, cancelSimRun := context.WithTimeout(context.Background(), 2*time.Minute)

	if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
		panic(err)
	}

	disconnections := sim.PeerEvents(
		context.Background(),
		sim.NodeIDs(),
		simulation.NewPeerEventsFilter().Type(p2p.PeerEventTypeDrop),
	)

	go func() {
		for d := range disconnections {
			log.Error("peer drop", "node", d.NodeID, "peer", d.Event.Peer)
			panic("unexpected disconnect")
			cancelSimRun()
		}
	}()

	return ctx, cancelSimRun
}

func TestNonExistingHashesWithServer(t *testing.T) {
	nodeCount, _, sim := setupSim(retrievalSimServiceMap)
	defer sim.Close()

	err := sim.UploadSnapshot(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		panic(err)
	}

	ctx, cancelSimRun := watchSim(sim)
	defer cancelSimRun()

	testDuration := 30 * time.Second
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		//check on the node's FileStore (netstore)
		id := sim.RandomUpNode().ID
		item, ok := sim.NodeItem(id, bucketKeyFileStore)
		if !ok {
			t.Fatalf("No filestore")
		}
		fileStore := item.(*storage.FileStore)
		fakeHash := storage.GenerateRandomChunk(1000).Address()
		fmt.Println(fakeHash)
		reader, _ := fileStore.Retrieve(context.TODO(), fakeHash)
		if _, err := reader.Size(ctx, nil); err != nil {
			fmt.Println("expected error for non-existing chunk")
		}

		time.Sleep(testDuration)

		return nil
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}

	evt := &simulations.Event{
		Type: EventTypeSimTerminated,
	}
	sim.Net.Events().Send(evt)

}

func TestSnapshotSyncWithServer(t *testing.T) {

	nodeCount, chunkCount, sim := setupSim(simServiceMap)
	defer sim.Close()

	log.Info("Initializing test config")

	conf := &synctestConfig{}
	//map of discover ID to indexes of chunks expected at that ID
	conf.idToChunksMap = make(map[discover.NodeID][]int)
	//map of overlay address to discover ID
	conf.addrToIDMap = make(map[string]discover.NodeID)
	//array where the generated chunk hashes will be stored
	conf.hashes = make([]storage.Address, 0)

	err := sim.UploadSnapshot(fmt.Sprintf("testing/snapshot_%d.json", nodeCount))
	if err != nil {
		panic(err)
	}

	ctx, cancelSimRun := watchSim(sim)
	defer cancelSimRun()

	//sim.PeerEvents(
	offeredHashesFilter := simulation.NewPeerEventsFilter().Type(p2p.PeerEventTypeMsgRecv).Protocol("stream").MsgCode(1)
	wantedFilter := simulation.NewPeerEventsFilter().Type(p2p.PeerEventTypeMsgRecv).Protocol("stream").MsgCode(2)
	deliveryFilter := simulation.NewPeerEventsFilter().Type(p2p.PeerEventTypeMsgRecv).Protocol("stream").MsgCode(6)
	eventC := sim.PeerEvents(ctx, sim.UpNodeIDs(), offeredHashesFilter, wantedFilter, deliveryFilter)

	quit := make(chan struct{})

	go func() {
		for e := range eventC {
			select {
			case <-quit:
				fmt.Println("quitting event loop")
				return
			default:
			}
			if e.Error != nil {
				t.Fatal(e.Error)
			}
			if *e.Event.MsgCode == uint64(1) {
				evt := &simulations.Event{
					Type: EventTypeChunkOffered,
					Node: sim.Net.GetNode(e.NodeID),
					//Data: fmt.Sprintf("%s", h),
				}
				sim.Net.Events().Send(evt)
			} else if *e.Event.MsgCode == uint64(2) {
				evt := &simulations.Event{
					Type: EventTypeChunkWanted,
					Node: sim.Net.GetNode(e.NodeID),
					//Data: fmt.Sprintf("%s", h),
				}
				sim.Net.Events().Send(evt)
			} else if *e.Event.MsgCode == uint64(6) {
				evt := &simulations.Event{
					Type: EventTypeChunkDelivered,
					Node: sim.Net.GetNode(e.NodeID),
					//Data: fmt.Sprintf("%s", h),
				}
				sim.Net.Events().Send(evt)
			}
		}
	}()
	result := runSim(conf, ctx, sim, chunkCount)

	evt := &simulations.Event{
		Type: EventTypeSimTerminated,
	}
	sim.Net.Events().Send(evt)

	if result.Error != nil {
		panic(result.Error)
	}
	close(quit)
	log.Info("Simulation ended")
}

/*
func decodeMsg(code int) error {
	val, ok := Spec.NewMsg(code)
	if !ok {
		return errorf("invalid msg code", "%v", msg.Code)
	}
	if err := rlp.DecodeBytes(wmsg.Payload, val); err != nil {
		return errorf(ErrDecode, "<= %v: %v", msg, err)
	}
}
*/
