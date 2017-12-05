// Copyright (c) 2017 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ifplugin

import (
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/vpp-agent/idxvpp/nametoidx"
	"github.com/ligato/vpp-agent/idxvpp/persist"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/bfd"
	intf "github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/stn"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/ifplugin/vppdump"
)

// Resync writes interfaces to the VPP
//
// - resyncs the VPP
// - temporary: (checks wether sw_if_indexes are not obsolate - this will be swapped with master ID)
// - deletes obsolate status data
func (plugin *InterfaceConfigurator) Resync(nbIfs []*intf.Interfaces_Interface) (errs []error) {
	plugin.Log.WithField("cfg", plugin).Debug("RESYNC Interface begin.")
	// Calculate and log interface resync
	defer func() {
		if plugin.Stopwatch != nil {
			plugin.Stopwatch.PrintLog()
		}
	}()

	// Dump current state of the VPP interfaces
	vppIfs, err := vppdump.DumpInterfaces(plugin.Log, plugin.vppCh, plugin.Stopwatch)
	if err != nil {
		return []error{err}
	}

	// Read persistent mapping
	persistentIfs := nametoidx.NewNameToIdx(plugin.Log, core.PluginName("defaultvppplugins-ifplugin"), "iface resync corr", nil)
	err = persist.Marshalling(plugin.ServiceLabel.GetAgentLabel(), plugin.swIfIndexes.GetMapping(), persistentIfs)
	if err != nil {
		return err
	}

	// Register default and ethernet interfaces
	configurableVppIfs := make(map[uint32]*vppdump.Interface, 0)
	for vppIfIdx, vppIf := range vppIfs {
		if vppIfIdx == 0 || vppIf.Type == intf.InterfaceType_ETHERNET_CSMACD {
			plugin.swIfIndexes.RegisterName(vppIf.VPPInternalName, vppIfIdx, &vppIf.Interfaces_Interface)
			continue
		}
		// fill map of all configurable VPP interfaces (skip default & ethernet interfaces)
		configurableVppIfs[vppIfIdx] = vppIf
	}

	// Handle case where persistent mapping is not available
	if len(persistentIfs.ListNames()) == 0 && len(configurableVppIfs) > 0 {
		plugin.Log.Debug("Persistent mapping for interfaces is empty, %v VPP interfaces is unknown", len(configurableVppIfs))
		// In such a case, there is nothing to correlate with. All existing interfaces will be removed
		var wasErr error
		for vppIfIdx, vppIf := range configurableVppIfs {
			// register interface before deletion (to keep state consistent)
			vppAgentIf := &vppIf.Interfaces_Interface
			vppAgentIf.Name = vppIf.VPPInternalName
			// todo plugin.swIfIndexes.RegisterName(vppAgentIf.Name, vppIfIdx, vppAgentIf)
			if err := plugin.deleteVPPInterface(vppAgentIf, vppIfIdx); err != nil {
				plugin.Log.Error("Error while removing interface: %v", err)
				wasErr = err
			}
		}
		// Configure NB interfaces
		for _, nbIf := range nbIfs {
			if err := plugin.ConfigureVPPInterface(nbIf); err != nil {
				plugin.Log.Error("Error while configuring interface: %v", err)
				wasErr = err
			}
		}
		return wasErr
	}

	// Find correlation between VPP, ETCD NB and persistent mapping. Update existing interfaces
	// and configure new ones
	var wasErr error
	plugin.Log.Debugf("Using persistent mapping to resync %v interfaces", len(configurableVppIfs))
	for _, nbIf := range nbIfs {
		persistIdx, _, found := persistentIfs.LookupIdx(nbIf.Name)
		if found {
			// last interface ID is known. Let's verify that there is an interface
			// with the same ID on the vpp
			var ifPresent bool
			var ifVppData *intf.Interfaces_Interface
			for vppIfIdx, vppIf := range configurableVppIfs {
				// Check at least interface type
				if vppIfIdx == persistIdx && vppIf.Type == nbIf.Type {
					ifPresent = true
					ifVppData = &vppIf.Interfaces_Interface
				}
			}
			if ifPresent && ifVppData != nil {
				// Interface exists on the vpp. Agent assumes that the the configured interface
				// correlates with the NB one (needs to be registered manually)
				plugin.swIfIndexes.RegisterName(nbIf.Name, persistIdx, nbIf)
				plugin.Log.Debugf("Registered existing interface %v with index %v", nbIf.Name, persistIdx)
				// todo calculate diff before modifying
				plugin.ModifyVPPInterface(nbIf, ifVppData)
			} else {
				// Interface exists in mapping but not in vpp.
				if err := plugin.ConfigureVPPInterface(nbIf); err != nil {
					plugin.Log.Error("Error while configuring interface: %v", err)
					wasErr = err
				}
			}
		} else {
			// a new interface (missing in persistent mapping)
			if err := plugin.ConfigureVPPInterface(nbIf); err != nil {
				plugin.Log.Error("Error while configuring interface: %v", err)
				wasErr = err
			}
		}
	}

	// Remove obsolete config
	for vppIfIdx, vppIf := range configurableVppIfs {
		// If interface index is not registered yet, remove it
		_, _, found := plugin.swIfIndexes.LookupName(vppIfIdx)
		if !found {
			// To keep interface state consistent, interface has to be
			// temporary registered before removal
			vppAgentIf := &vppIf.Interfaces_Interface
			vppAgentIf.Name = vppIf.VPPInternalName
			// todo plugin.swIfIndexes.RegisterName(vppAgentIf.Name, vppIfIdx, vppAgentIf)
			if err := plugin.deleteVPPInterface(vppAgentIf, vppIfIdx); err != nil {
				plugin.Log.Error("Error while removing interface: %v", err)
				wasErr = err
			}
			plugin.Log.Debugf("Removed unknown interface with index %v", vppIfIdx)
		}
	}

	plugin.Log.WithField("cfg", plugin).Debug("RESYNC Interface end.")

	return wasErr
}

// VerifyVPPConfigPresence dumps VPP interface configuration on the vpp. If there are any interfaces configured (except
// the local0), it returns false (do not interrupt the resto of the resync), otherwise returns true
func (plugin *InterfaceConfigurator) VerifyVPPConfigPresence(nbIfaces []*intf.Interfaces_Interface) bool {
	plugin.Log.WithField("cfg", plugin).Debug("RESYNC Interface begin.")
	// notify that the resync should be stopped
	var stop bool

	// Step 0: Dump actual state of the VPP
	vppIfaces, err := vppdump.DumpInterfaces(plugin.Log, plugin.vppCh, plugin.Stopwatch)
	if err != nil {
		return stop
	}

	// The strategy is optimize-cold-start, so look over all dumped VPP interfaces and check for the configured ones
	// (leave out the local0). If there are any other interfaces, return true (resync will continue).
	// If not, return a false flag which cancels the VPP resync operation.
	plugin.Log.Info("optimize-cold-start VPP resync strategy chosen, resolving...")
	if len(vppIfaces) == 0 {
		stop = true
		plugin.Log.Infof("...VPP resync interrupted assuming there is no configuration on the VPP (no interface was found)")
		return stop
	}
	// in interface exists, try to find local0 interface (index 0)
	_, ok := vppIfaces[0]
	// in case local0 is the only interface on the vpp, stop the resync
	if len(vppIfaces) == 1 && ok {
		stop = true
		plugin.Log.Infof("...VPP resync interrupted assuming there is no configuration on the VPP (only local0 was found)")
		return stop
	}
	// otherwise continue normally
	plugin.Log.Infof("... VPP configuration found, continue with VPP resync")

	return stop
}

// ResyncSession writes BFD sessions to the empty VPP
func (plugin *BFDConfigurator) ResyncSession(bfds []*bfd.SingleHopBFD_Session) error {
	plugin.Log.WithField("cfg", plugin).Debug("RESYNC BFD Session begin.")
	// Calculate and log bfd resync
	defer func() {
		if plugin.Stopwatch != nil {
			plugin.Stopwatch.PrintLog()
		}
	}()

	// lookup BFD sessions
	err := plugin.LookupBfdSessions()
	if err != nil {
		return err
	}

	var wasError error

	// create BFD sessions
	for _, bfdSession := range bfds {
		err = plugin.ConfigureBfdSession(bfdSession)
		if err != nil {
			wasError = err
		}
	}

	plugin.Log.WithField("cfg", plugin).Debug("RESYNC BFD Session end. ", wasError)

	return wasError
}

// ResyncAuthKey writes BFD keys to the empty VPP
func (plugin *BFDConfigurator) ResyncAuthKey(bfds []*bfd.SingleHopBFD_Key) error {
	plugin.Log.WithField("cfg", plugin).Debug("RESYNC BFD Keys begin.")
	// Calculate and log bfd resync
	defer func() {
		if plugin.Stopwatch != nil {
			plugin.Stopwatch.PrintLog()
		}
	}()

	// lookup BFD auth keys
	err := plugin.LookupBfdKeys()
	if err != nil {
		return err
	}

	var wasError error

	// create BFD auth keys
	for _, bfdKey := range bfds {
		err = plugin.ConfigureBfdAuthKey(bfdKey)
		if err != nil {
			wasError = err
		}
	}

	plugin.Log.WithField("cfg", plugin).Debug("RESYNC BFD Keys end. ", wasError)

	return wasError
}

// ResyncEchoFunction writes BFD echo function to the empty VPP
func (plugin *BFDConfigurator) ResyncEchoFunction(bfds []*bfd.SingleHopBFD_EchoFunction) error {
	return nil
}

// Resync writes stn rule to the the empty VPP
func (plugin *StnConfigurator) Resync(stnRules []*stn.StnRule) error {
	plugin.Log.WithField("cfg", plugin).Debug("RESYNC stn rules begin. ")
	// Calculate and log stn rules resync
	defer func() {
		if plugin.Stopwatch != nil {
			plugin.Stopwatch.PrintLog()
		}
	}()

	var wasError error
	if len(stnRules) > 0 {
		for _, rule := range stnRules {
			wasError = plugin.Add(rule)
		}
	}
	plugin.Log.WithField("cfg", plugin).Debug("RESYNC stn rules end. ", wasError)
	return wasError
}
