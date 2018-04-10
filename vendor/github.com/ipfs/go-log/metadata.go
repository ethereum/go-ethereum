package log

import (
	"encoding/json"
	"errors"
	"reflect"
)

// Metadata is a convenience type for generic maps
type Metadata map[string]interface{}

// DeepMerge merges the second Metadata parameter into the first.
// Nested Metadata are merged recursively. Primitives are over-written.
func DeepMerge(b, a Metadata) Metadata {
	out := Metadata{}
	for k, v := range b {
		out[k] = v
	}
	for k, v := range a {

		maybe, err := Metadatify(v)
		if err != nil {
			// if the new value is not meta. just overwrite the dest vaue
			if out[k] != nil {
				log.Errorf("Over writting key: %s, old: %s, new: %s", k, out[k], v)
			}
			out[k] = v
			continue
		}

		// it is meta. What about dest?
		outv, exists := out[k]
		if !exists {
			// the new value is meta, but there's no dest value. just write it
			out[k] = v
			continue
		}

		outMetadataValue, err := Metadatify(outv)
		if err != nil {
			// the new value is meta and there's a dest value, but the dest
			// value isn't meta. just overwrite
			out[k] = v
			continue
		}

		// both are meta. merge them.
		out[k] = DeepMerge(outMetadataValue, maybe)
	}
	return out
}

// Loggable implements the Loggable interface.
func (m Metadata) Loggable() map[string]interface{} {
	// NB: method defined on value to avoid de-referencing nil Metadata
	return m
}

// JsonString returns the marshaled JSON string for the metadata.
func (m Metadata) JsonString() (string, error) {
	// NB: method defined on value
	b, err := json.Marshal(m)
	return string(b), err
}

// Metadatify converts maps into Metadata.
func Metadatify(i interface{}) (Metadata, error) {
	value := reflect.ValueOf(i)
	if value.Kind() == reflect.Map {
		m := map[string]interface{}{}
		for _, k := range value.MapKeys() {
			m[k.String()] = value.MapIndex(k).Interface()
		}
		return Metadata(m), nil
	}
	return nil, errors.New("is not a map")
}
