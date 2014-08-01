## Reactor

Reactor is the internal broadcast engine that allows components to be notified of ethereum stack events such as finding new blocks or change in state.
Event notification is handled via subscription:

    var blockChan = make(chan ethreact.Event, 10)
    reactor.Subscribe("newBlock", blockChan)

ethreact.Event broadcast on the channel are 

    type Event struct {
        Resource interface{}
        Name     string
    } 

Resource is polimorphic depending on the event type and should be typecast before use, e.g:

    b := <-blockChan:
    block := b.Resource.(*ethchain.Block)

Events are guaranteed to be broadcast in order but the broadcast never blocks or leaks which means while the subscribing event channel is blocked (e.g., full if buffered) further messages will be skipped. 

The engine allows arbitrary events to be posted and subscribed to. 

    ethereum.Reactor().Post("newBlock", newBlock)


    