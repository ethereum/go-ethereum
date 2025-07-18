// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/params/forks"
)

// Activations contains the block numbers/timestamps where hard forks activate.
type Activations map[forks.Fork]uint64

// Config2 represents the chain configuration.
type Config2 struct {
	activation Activations
	param      map[int]any
}

func NewConfig2(activations Activations, param ...ParamValue) *Config2 {
	cfg := &Config2{
		activation: maps.Clone(activations),
		param:      make(map[int]any, len(param)),
	}
	cfg.activation[forks.Frontier] = 0
	for _, pv := range param {
		cfg.param[pv.id] = pv.value
	}
	return cfg
}

// Active reports whether the given fork is active for a block number/time.
func (cfg *Config2) Active(f forks.Fork, block, timestamp uint64) bool {
	activation, ok := cfg.activation[f]
	if f.BlockBased() {
		return ok && block >= activation
	}
	return ok && timestamp >= activation
}

// ActiveAtBlock reports whether the given fork is active for a block number/time.
func (cfg *Config2) ActiveAtBlock(f forks.Fork, block *big.Int) bool {
	if !f.BlockBased() {
		panic(fmt.Sprintf("fork %v has time-based scheduling", f))
	}
	activation, ok := cfg.activation[f]
	return ok && block.Uint64() >= activation
}

// Activation returns the activation block/number of a fork.
func (cfg *Config2) Activation(f forks.Fork) (uint64, bool) {
	a, ok := cfg.activation[f]
	return a, ok
}

// Scheduled says whether the fork is scheduled at all.
func (cfg *Config2) Scheduled(f forks.Fork) bool {
	_, ok := cfg.activation[f]
	return ok
}

// SetActivations returns a new configuration with the given forks activated.
func (cfg *Config2) SetActivations(forks Activations) *Config2 {
	newA := maps.Clone(cfg.activation)
	maps.Copy(newA, forks)
	return &Config2{activation: newA, param: cfg.param}
}

// SetParam returns a new configuration with the given parameter values set.
func (cfg *Config2) SetParam(param ...ParamValue) *Config2 {
	cpy := &Config2{activation: cfg.activation, param: maps.Clone(cfg.param)}
	for _, pv := range param {
		cpy.param[pv.id] = pv.value
	}
	return cpy
}

// LatestFork returns the latest time-based fork that would be active for the given time.
func (cfg *Config2) LatestFork(time uint64) forks.Fork {
	londonBlock := cfg.activation[forks.London]
	for _, f := range slices.Backward(forks.CanonOrder) {
		if f.BlockBased() {
			break
		}
		if cfg.Active(f, londonBlock, time) {
			return f
		}
	}
	return forks.Paris
}

// MarshalJSON encodes the config as JSON.
func (cfg *Config2) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	// params
	for id, value := range cfg.param {
		info, ok := paramRegistry[id]
		if !ok {
			panic(fmt.Sprintf("unknown chain parameter id %v", id))
		}
		m[info.name] = value
	}
	// forks
	for f, act := range cfg.activation {
		var name string
		if f.BlockBased() {
			name = fmt.Sprintf("%sBlock", strings.ToLower(name))
		} else {
			name = fmt.Sprintf("%sTime", strings.ToLower(name))
		}
		m[name] = act
	}
	return json.Marshal(m)
}

// MarshalJSON encodes the config as JSON.
func (cfg *Config2) UnmarshalJSON(input []byte) error {
	dec := json.NewDecoder(bytes.NewReader(input))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if tok != json.Delim('{') {
		return fmt.Errorf("expected JSON object for chain configuration")
	}
	// Now we're in the object.
	newcfg := Config2{
		activation: make(Activations),
	}
	for {
		tok, err = dec.Token()
		if tok == json.Delim('}') {
			break
		} else if key, ok := tok.(string); ok {
			if strings.HasSuffix(key, "Block") || strings.HasSuffix(key, "Time") {
				err = newcfg.decodeActivation(key, dec)
			} else {
				err = newcfg.decodeParameter(key, dec)
			}
		}
		if err != nil {
			return err
		}
	}

	*cfg = newcfg
	return nil
}

func (cfg *Config2) decodeActivation(key string, dec *json.Decoder) error {
	var num uint64
	if err := dec.Decode(&num); err != nil {
		return err
	}

	var f forks.Fork
	name, ok := strings.CutSuffix(key, "Block")
	if ok {
		f, ok = forks.ByName(name)
		if !ok || !f.BlockBased() {
			return fmt.Errorf("unknown block-based fork %q", name)
		}
	} else if name, ok = strings.CutSuffix(key, "Time"); ok {
		f, ok = forks.ByName(name)
		if !ok || f.BlockBased() {
			return fmt.Errorf("unknown time-based fork %q", name)
		}
	}
	cfg.activation[f] = num
	return nil
}

func (cfg *Config2) decodeParameter(key string, dec *json.Decoder) error {
	id, ok := paramRegistryByName[key]
	if !ok {
		return fmt.Errorf("unknown chain parameter %q", key)
	}
	v := paramRegistry[id].new()
	if err := dec.Decode(v); err != nil {
		return err
	}
	cfg.param[id] = v
	return nil
}

// Validate checks the configuration to ensure forks are scheduled in order,
// and required settings are present.
func (cfg *Config2) Validate() error {
	sanityCheckCanonOrder()

	// Check forks.
	lastFork := forks.CanonOrder[0]
	for _, f := range forks.CanonOrder[1:] {
		act := "timestamp"
		if f.BlockBased() {
			act = "block"
		}

		switch {
		// Non-optional forks must all be present in the chain config up to the last defined fork.
		case !cfg.Scheduled(lastFork) && cfg.Scheduled(f):
			return fmt.Errorf("unsupported fork ordering: %v not enabled, but %v enabled at %s %v", lastFork, f, act, cfg.activation[f])

		// Fork (whether defined by block or timestamp) must follow the fork definition sequence.
		case cfg.Scheduled(lastFork) && cfg.Scheduled(f):
			// Timestamp based forks can follow block based ones, but not the other way around.
			if !lastFork.BlockBased() && f.BlockBased() {
				return fmt.Errorf("unsupported fork ordering: %v used timestamp ordering, but %v reverted to block ordering", lastFork, f)
			}
			if lastFork.BlockBased() == f.BlockBased() && cfg.activation[lastFork] > cfg.activation[f] {
				return fmt.Errorf("unsupported fork ordering: %v enabled at %s %v, but %v enabled at %s %v", lastFork, act, cfg.activation[lastFork], f, act, cfg.activation[f])
			}
		}

		// If it was optional and not set, then ignore it.
		if !f.Optional() || cfg.Scheduled(f) {
			lastFork = f
		}
	}

	// Check parameters.
	for id, info := range paramRegistry {
		v, isSet := cfg.param[id]
		if !isSet {
			if !info.optional {
				return fmt.Errorf("required chain parameter %s is not set", info.name)
			}
			v = info.defaultValue
		}
		if err := info.validate(v, cfg); err != nil {
			return fmt.Errorf("invalid %s: %w", info.name, err)
		}
	}

	return nil
}

// CheckCompatible validates chain configuration changes.
// This called before applying changes to the 'stored configuration', the config
// which is held in the database. The given block number and time represent the current head
// of the chain in that database.
//
// An error is returned when the new configuration attempts to schedule a fork below the
// current chain head. The error contains enough information to rewind the chain to a
// point where the new config can be applied safely.
func (c *Config2) CheckCompatible(newcfg *Config2, blocknum uint64, time uint64) *ConfigCompatError {
	sanityCheckCanonOrder()

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	bhead, btime := blocknum, time
	for {
		err := c.checkCompatible(newcfg, bhead, btime)
		if err == nil || (lasterr != nil && err.RewindToBlock == lasterr.RewindToBlock && err.RewindToTime == lasterr.RewindToTime) {
			break
		}
		lasterr = err

		if err.RewindToTime > 0 {
			btime = err.RewindToTime
		} else {
			bhead = err.RewindToBlock
		}
	}
	return lasterr
}

// checkCompatible checks config compatibility at a specific block height.
func (cfg *Config2) checkCompatible(newcfg *Config2, num uint64, time uint64) *ConfigCompatError {
	incompatible := func(f forks.Fork) bool {
		return (cfg.Active(f, num, time) || newcfg.Active(f, num, time)) && !activationEqual(f, cfg, newcfg)
	}

	for _, f := range forks.CanonOrder[1:] {
		if incompatible(f) {
			if f.BlockBased() {
				return newBlockCompatError2(fmt.Sprintf("%v fork block", f), f, cfg, newcfg)
			}
			return newTimestampCompatError2("%v fork timestamp", f, cfg, newcfg)
		}
	}

	if cfg.Active(forks.DAO, num, time) && DAOForkSupport.Get(cfg) != DAOForkSupport.Get(newcfg) {
		return newBlockCompatError2("DAO fork support flag", forks.DAO, cfg, newcfg)
	}
	if cfg.Active(forks.TangerineWhistle, num, time) && !configBlockEqual(ChainID.Get(cfg), ChainID.Get(newcfg)) {
		return newBlockCompatError2("EIP158 chain ID", forks.TangerineWhistle, cfg, newcfg)
	}

	return nil
}

func newBlockCompatError2(what string, f forks.Fork, storedcfg, newcfg *Config2) *ConfigCompatError {
	err := &ConfigCompatError{What: what}
	if storedcfg.Scheduled(f) {
		err.StoredBlock = new(big.Int).SetUint64(storedcfg.activation[f])
	}
	if newcfg.Scheduled(f) {
		err.NewBlock = new(big.Int).SetUint64(newcfg.activation[f])
	}
	// Need to rewind to one block before the earliest possible activation.
	rew, _ := minActivation(f, storedcfg, newcfg)
	if rew > 0 {
		err.RewindToBlock = rew - 1
	}
	return err
}

func newTimestampCompatError2(what string, f forks.Fork, storedcfg, newcfg *Config2) *ConfigCompatError {
	err := &ConfigCompatError{What: what}
	if storedcfg.Scheduled(f) {
		t := storedcfg.activation[f]
		err.StoredTime = &t
	}
	if newcfg.Scheduled(f) {
		t := newcfg.activation[f]
		err.NewTime = &t
	}
	// Need to rewind to before the earliest possible activation.
	rew, _ := minActivation(f, storedcfg, newcfg)
	if rew > 0 {
		err.RewindToTime = rew - 1
	}
	return err
}

// minActivation the earliest possible activation block/time for the given fork
// across two configurations.
func minActivation(f forks.Fork, cfg1, cfg2 *Config2) (act uint64, scheduled bool) {
	switch {
	case !cfg1.Scheduled(f):
		act, scheduled = cfg2.activation[f]
	case !cfg2.Scheduled(f):
		act, scheduled = cfg1.activation[f]
	default:
		act = min(cfg1.activation[f], cfg2.activation[f])
		scheduled = true
	}
	return
}

// activationEqual checks whether a fork has the same activation two configurations.
// Note this also returns true when it isn't scheduled at all.
func activationEqual(f forks.Fork, cfg1, cfg2 *Config2) bool {
	return cfg1.Scheduled(f) == cfg2.Scheduled(f) && cfg1.activation[f] == cfg2.activation[f]
}

// sanityCheckCanonOrder verifies forks.CanonOrder is defined sensibly.
// This exists to ensure library code doesn't mess with this slice in an incompatible way.
func sanityCheckCanonOrder() {
	if len(forks.CanonOrder) == 0 {
		panic("forks.CanonOrder is empty")
	}
	if forks.CanonOrder[0] != forks.Frontier {
		panic("forks.CanonOrder must start with Frontier")
	}
}

func cloneBig(x *big.Int) *big.Int {
	if x == nil {
		return nil
	}
	return new(big.Int).Set(x)
}
