// Copyright 2024 the libevm authors.
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

	"github.com/ava-labs/libevm/libevm/pseudo"
)

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = (*ChainConfig)(nil)

// chainConfigWithoutMethods avoids infinite recurion into
// [ChainConfig.UnmarshalJSON].
type chainConfigWithoutMethods ChainConfig

// chainConfigWithExportedExtra supports JSON (un)marshalling of a [ChainConfig]
// while exposing the `extra` field as the "extra" JSON key.
type chainConfigWithExportedExtra struct {
	*chainConfigWithoutMethods              // embedded to achieve regular JSON unmarshalling
	Extra                      *pseudo.Type `json:"extra"` // `c.extra` is otherwise unexported
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (c *ChainConfig) UnmarshalJSON(data []byte) error {
	switch reg := registeredExtras; {
	case reg != nil && !reg.reuseJSONRoot:
		return c.unmarshalJSONWithExtra(data)

	case reg != nil && reg.reuseJSONRoot: // although the latter is redundant, it's clearer
		c.extra = reg.newChainConfig()
		if err := json.Unmarshal(data, c.extra); err != nil {
			c.extra = nil
			return err
		}
		fallthrough // Important! We've only unmarshalled the extra field.
	default: // reg == nil
		return json.Unmarshal(data, (*chainConfigWithoutMethods)(c))
	}
}

// unmarshalJSONWithExtra unmarshals JSON under the assumption that the
// registered [Extras] payload is in the JSON "extra" key. All other
// unmarshalling is performed as if no [Extras] were registered.
func (c *ChainConfig) unmarshalJSONWithExtra(data []byte) error {
	cc := &chainConfigWithExportedExtra{
		chainConfigWithoutMethods: (*chainConfigWithoutMethods)(c),
		Extra:                     registeredExtras.newChainConfig(),
	}
	if err := json.Unmarshal(data, cc); err != nil {
		return err
	}
	c.extra = cc.Extra
	return nil
}

// MarshalJSON implements the [json.Marshaler] interface.
func (c *ChainConfig) MarshalJSON() ([]byte, error) {
	switch reg := registeredExtras; {
	case reg == nil:
		return json.Marshal((*chainConfigWithoutMethods)(c))

	case !reg.reuseJSONRoot:
		return c.marshalJSONWithExtra()

	default: // reg.reuseJSONRoot == true
		// The inverse of reusing the JSON root is merging two JSON buffers,
		// which isn't supported by the native package. So we use
		// map[string]json.RawMessage intermediates.
		geth, err := toJSONRawMessages((*chainConfigWithoutMethods)(c))
		if err != nil {
			return nil, err
		}
		extra, err := toJSONRawMessages(c.extra)
		if err != nil {
			return nil, err
		}

		for k, v := range extra {
			if _, ok := geth[k]; ok {
				return nil, fmt.Errorf("duplicate JSON key %q in both %T and registered extra", k, c)
			}
			geth[k] = v
		}
		return json.Marshal(geth)
	}
}

// marshalJSONWithExtra is the inverse of unmarshalJSONWithExtra().
func (c *ChainConfig) marshalJSONWithExtra() ([]byte, error) {
	cc := &chainConfigWithExportedExtra{
		chainConfigWithoutMethods: (*chainConfigWithoutMethods)(c),
		Extra:                     c.extra,
	}
	return json.Marshal(cc)
}

func toJSONRawMessages(v any) (map[string]json.RawMessage, error) {
	buf, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	msgs := make(map[string]json.RawMessage)
	if err := json.Unmarshal(buf, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}
