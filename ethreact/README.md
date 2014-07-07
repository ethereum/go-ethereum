# ethreact

ethereum event reactor. Component of the ethereum stack.
various events like state change on an account or new block found are broadcast to subscribers.
Broadcasting to subscribers is running on its own routine and globally order preserving.

## Clients
### subscribe

    eventChannel := make(chan ethreact.Event)
    reactor.Subscribe(event, eventChannel)

The same channel can be subscribed to multiple events but only once for each event. In order to allow order of events to be preserved, broadcast of events is synchronous within the main broadcast loop. Therefore any blocking subscriber channels will be skipped, i.e. missing broadcasting events while they are blocked.

### unsubscribe

    reactor.Unsubscribe(event, eventChannel)

### Processing events

event.Resource is of type interface{}. The actual type of event.Resource depends on event.Name and may need to be cast for processing.

    var event ethreact.Event
    for {
        select {
        case event = <-eventChannel:
            processTransaction(event.Resource.(Transaction))
        }
    }

## Broadcast 

    reactor := ethreact.New()
    reactor.Start()
    reactor.Post(name, resource)
    reactor.Flush() // wait till all broadcast messages are dispatched
    reactor.Stop() // stop the main broadcast loop immediately (even if there are unbroadcast events left)



