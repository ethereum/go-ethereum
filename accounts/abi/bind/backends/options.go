// Copyright 2015 The go-ethereum Authors
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

package backends

type cloneOnCall struct {
	Option
}

// CloneVMConfigOnCall instructs the SimulatedBackend to clone its Blockchain's
// vm.Config when CallContract() is invoked. Historically, the Config was always
// overrideen and this remains the default for backwards compatibility. The
// Config's NoBaseFee field is always overridden to true.
func CloneVMConfigOnCall() Option {
	return cloneOnCall{}
}

type cloneOnPendingCall struct {
	Option
}

// CloneVMConfigOnPendingCall instructs the SimulatedBackend to clone its
// Blockchain's vm.Config when PendingCallContract() is invoked. Historically,
// the Config was always overrideen and this remains the default for backwards
// compatibility. The Config's NoBaseFee field is always overridden to true.
func CloneVMConfigOnPendingCall() Option {
	return cloneOnPendingCall{}
}

type cloneOnEstimateGas struct {
	Option
}

// CloneVMConfigOnEstimateGas instructs the SimulatedBackend to clone its
// Blockchain's vm.Config when EstimateGas() is invoked. Historically, the
// Config was always overrideen and this remains the default for backwards
// compatibility. The Config's NoBaseFee field is always overridden to true.
func CloneVMConfigOnEstimateGas() Option {
	return cloneOnEstimateGas{}
}

type alwaysCloneVMConfig struct {
	Option
}

// AlwaysCloneVMConfig is equivalent to passing all of the CloneVMConfigOn*()
// Options to SimulatedBackend constructors.
func AlwaysCloneVMConfig() Option {
	return alwaysCloneVMConfig{}
}
