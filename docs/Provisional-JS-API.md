The *provisional* JavaScript API is a purposed API for all things JavaScript. JavaScript technologies can be embedded within Qt(QML) technologies, local web and remote web and therefor the purposed API is written in a ASYNC fashion so that it may be used across all implementations. Hereby it should be known that all functions, unless explicitly specified, take a callback as last function argument which will be called when the operation has been completed.

Please note that the provisional JavaScript API tries to leverage existing JS idioms as much as possible.

## General API

* `getBlock (number or string)`
    Retrieves a block by either the address or the number. If supplied with a string it will assume address, number otherwise.
* `transact (sec, recipient, value, gas, gas price, data)`
    Creates a new transaction using your current key.
* `create (sec, value, gas, gas price, init, body)`
    Creates a new contract using your current key.
* `getKey (none)`
    Retrieves your current key in hex format.
* `getStorage (object address, storage address)`
    Retrieves the storage address of the given object.
* `getBalance (object address)`
    Retrieves the balance at the current address
* `watch (string [, string])`
    Watches for changes on a specific address' state object such as state root changes or value changes.
* `disconnect (string [, string])`
    Disconnects from a previous `watched` address.

## Events

The provisional JavaScript API exposes certain events through a basic eventing mechanism inspired by jQuery.

* `on (event)`
    Subscribe to event which will be called whenever an event of type <event> is received.
* `off (event)`
    Unsubscribe to the given event
* `trigger (event, data)`
    Trigger event of type <event> with the given data. **note:** This function does not take a callback function.
    
### Event Types

All events are written in camel cased style beginning with a lowercase letter. Subevents are denoted by a colon `:`.

* `block:new`
    Fired when a new valid block has been found on the wire. The attached value of this call is a block.
* `object:changed`
    Fired when a watched address, specified through `watch`, changes in value.