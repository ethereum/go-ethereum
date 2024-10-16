// Copyright 2024 The go-ethereum Authors
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

package exex

import (
	"errors"
	"sort"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Plugins returns a list of all registered plugins to generate CLI flags.
func (reg *registry) Plugins() []string {
	plugins := make([]string, 0, len(reg.pluginsMakersV1))
	for name := range reg.pluginsMakersV1 {
		plugins = append(plugins, name)
	}
	sort.Strings(plugins)
	return plugins
}

// Instantiate constructs an execution extension plugin from a unique name.
func (reg *registry) Instantiate(name string, userconf string) error {
	// Try instantiating a V1 plugin
	if constructor, ok := globalRegistry.pluginsMakersV1[name]; ok {
		plugin, err := constructor(&ConfigV1{
			Logger: log.New("exex", name),
			User:   userconf,
		})
		if err != nil {
			return err
		}
		globalRegistry.pluginsV1[name] = plugin
		return nil
	}
	// No plugins matched across any versions, return a failure
	return errors.New("not found")
}

// TriggerInitHook triggers the OnInit hook in exex plugins.
func (reg *registry) TriggerInitHook(chain Chain) {
	for _, plugin := range globalRegistry.pluginsV1 {
		if plugin.OnInit != nil {
			plugin.OnInit(chain)
		}
	}
}

// TriggerCloseHook triggers the OnClose hook in exex plugins.
func (reg *registry) TriggerCloseHook() {
	for _, plugin := range globalRegistry.pluginsV1 {
		if plugin.OnClose != nil {
			plugin.OnClose()
		}
	}
}

// TriggerHeadHook triggers the OnHead hook in exex plugins.
func (reg *registry) TriggerHeadHook(head *types.Header) {
	for _, plugin := range globalRegistry.pluginsV1 {
		if plugin.OnHead != nil {
			plugin.OnHead(head)
		}
	}
}

// TriggerReorgHook triggers the OnReorg hook in exex plugins.
func (reg *registry) TriggerReorgHook(headers []*types.Header, revert bool) {
	for _, plugin := range globalRegistry.pluginsV1 {
		if plugin.OnReorg != nil {
			plugin.OnReorg(headers, revert)
		}
	}
}
