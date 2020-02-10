package keystore

import "reflect"

// DBKeyStoreType is the reflect type of a keystore backend.
var DBKeyStoreType = reflect.TypeOf(&keyStoreDB{})

type keyStoreDB struct {
}
