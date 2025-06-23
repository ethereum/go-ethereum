// Copyright 2024-2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package params

import (
	"encoding/json"
	"fmt"
)

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = (*ChainConfig)(nil)

// chainConfigWithoutMethods avoids infinite recursion into
// [ChainConfig.UnmarshalJSON].
type chainConfigWithoutMethods ChainConfig

// UnmarshalJSON implements the [json.Unmarshaler] interface. If extra payloads
// were registered, UnmarshalJSON decodes data as described by [Extras] and
// [RegisterExtras] otherwise it unmarshals directly into c as if ChainConfig
// didn't implement json.Unmarshaler.
func (c *ChainConfig) UnmarshalJSON(data []byte) (err error) {
	if !registeredExtras.Registered() {
		return json.Unmarshal(data, (*chainConfigWithoutMethods)(c))
	}
	ec := registeredExtras.Get()
	c.extra = ec.newChainConfig()
	return UnmarshalChainConfigJSON(data, c, c.extra, ec.reuseJSONRoot)
}

// UnmarshalChainConfigJSON is equivalent to [ChainConfig.UnmarshalJSON]
// had [Extras] with `C` been registered, but without the need to call
// [RegisterExtras]. The `extra` argument MUST NOT be nil.
func UnmarshalChainConfigJSON[C any](data []byte, config *ChainConfig, extra *C, reuseJSONRoot bool) (err error) {
	if extra == nil {
		return fmt.Errorf("%T argument is nil; use %T.UnmarshalJSON() directly", extra, config)
	}

	if reuseJSONRoot {
		if err := json.Unmarshal(data, (*chainConfigWithoutMethods)(config)); err != nil {
			return fmt.Errorf("decoding JSON into %T: %s", config, err)
		}
		if err := json.Unmarshal(data, extra); err != nil {
			return fmt.Errorf("decoding JSON into %T: %s", extra, err)
		}
		return nil
	}

	combined := struct {
		*chainConfigWithoutMethods
		Extra *C `json:"extra"`
	}{
		(*chainConfigWithoutMethods)(config),
		extra,
	}
	if err := json.Unmarshal(data, &combined); err != nil {
		return fmt.Errorf(`decoding JSON into combination of %T and %T (as "extra" key): %s`, config, extra, err)
	}
	return nil
}

// MarshalJSON implements the [json.Marshaler] interface.
// If extra payloads were registered, MarshalJSON encodes JSON as
// described by [Extras] and [RegisterExtras] otherwise it marshals
// `c` as if ChainConfig didn't implement json.Marshaler.
func (c *ChainConfig) MarshalJSON() ([]byte, error) {
	if !registeredExtras.Registered() {
		return json.Marshal((*chainConfigWithoutMethods)(c))
	}
	ec := registeredExtras.Get()
	return MarshalChainConfigJSON(*c, c.extra, ec.reuseJSONRoot)
}

// MarshalChainConfigJSON is equivalent to [ChainConfig.MarshalJSON]
// had [Extras] with `C` been registered, but without the need to
// call [RegisterExtras].
func MarshalChainConfigJSON[C any](config ChainConfig, extra C, reuseJSONRoot bool) (data []byte, err error) {
	if !reuseJSONRoot {
		jsonExtra := struct {
			ChainConfig
			Extra C `json:"extra,omitempty"`
		}{
			config,
			extra,
		}
		data, err = json.Marshal(jsonExtra)
		if err != nil {
			return nil, fmt.Errorf(`encoding combination of %T and %T (as "extra" key) to JSON: %s`, config, extra, err)
		}
		return data, nil
	}

	// The inverse of reusing the JSON root is merging two JSON buffers,
	// which isn't supported by the native package. So we use
	// map[string]json.RawMessage intermediates.
	// Note we cannot encode a combined struct directly because of the extra
	// type generic nature which cannot be embedded in such a combined struct.
	configJSONRaw, err := toJSONRawMessages((chainConfigWithoutMethods)(config))
	if err != nil {
		return nil, fmt.Errorf("converting config to JSON raw messages: %s", err)
	}
	extraJSONRaw, err := toJSONRawMessages(extra)
	if err != nil {
		return nil, fmt.Errorf("converting extra config to JSON raw messages: %s", err)
	}

	for k, v := range extraJSONRaw {
		_, ok := configJSONRaw[k]
		if ok {
			return nil, fmt.Errorf("duplicate JSON key %q in ChainConfig and extra %T", k, extra)
		}
		configJSONRaw[k] = v
	}
	return json.Marshal(configJSONRaw)
}

func toJSONRawMessages(v any) (map[string]json.RawMessage, error) {
	buf, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encoding %T: %s", v, err)
	}
	msgs := make(map[string]json.RawMessage)
	if err := json.Unmarshal(buf, &msgs); err != nil {
		return nil, fmt.Errorf("decoding JSON encoding of %T into %T: %s", v, msgs, err)
	}
	return msgs, nil
}
