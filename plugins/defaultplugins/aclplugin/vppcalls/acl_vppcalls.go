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

package vppcalls

import (
	"fmt"
	"net"
	"strings"
	"time"

	"git.fd.io/govpp.git/api"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/logrus"
	"github.com/ligato/cn-infra/logging/measure"
	acl_api "github.com/ligato/vpp-agent/plugins/defaultplugins/common/bin_api/acl"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/acl"
)

// Protocol types that can occur in ACLs
const (
	ICMPv4Proto = 1
	TCPProto    = 6
	UDPProto    = 17
	ICMPv6Proto = 58
)

// AddIPAcl create new L3/4 ACL. Input index == 0xffffffff, VPP provides index in reply.
func AddIPAcl(rules []*acl.AccessLists_Acl_Rule, aclName string, log logging.Logger,
	vppChannel *api.Channel, stopwatch *measure.Stopwatch) (uint32, error) {
	defer func(t time.Time) {
		stopwatch.TimeLog(acl_api.ACLAddReplace{}).LogTimeEntry(time.Since(t))
	}(time.Now())

	// Prepare Ip rules
	aclIPRules, err := transformACLIpRules(rules)
	if err != nil {
		return 0, err
	}
	if len(aclIPRules) != 0 {
		req := &acl_api.ACLAddReplace{
			ACLIndex: 0xffffffff, // to make new Entry
			Count:    uint32(len(aclIPRules)),
			Tag:      []byte(aclName),
			R:        aclIPRules,
		}

		reply := &acl_api.ACLAddReplaceReply{}

		if err = vppChannel.SendRequest(req).ReceiveReply(reply); err != nil {
			return 0, fmt.Errorf("failed to write ACL %v: %v", aclName, err)
		}
		if reply.Retval != 0 {
			return 0, fmt.Errorf("error %v while writing ACL %v to VPP", reply.Retval, aclName)
		}
		log.Infof("%v Ip ACL rule(s) written for ACL %v with index %v", len(aclIPRules), aclName, reply.ACLIndex)

		return reply.ACLIndex, nil
	}

	return 0, fmt.Errorf("no rules found for ACL %v", aclName)
}

// AddMacIPAcl creates new L2 MAC IP ACL. VPP provides index in reply.
func AddMacIPAcl(rules []*acl.AccessLists_Acl_Rule, aclName string, log logging.Logger,
	vppChannel *api.Channel, stopwatch *measure.Stopwatch) (uint32, error) {
	defer func(t time.Time) {
		stopwatch.TimeLog(acl_api.MacipACLAdd{}).LogTimeEntry(time.Since(t))
	}(time.Now())

	// Prepare MAc Ip rules
	aclMacIPRules, err := transformACLMacIPRules(rules)
	if err != nil {
		return 0, err
	}
	if len(aclMacIPRules) == 0 {
		log.Debugf("No Mac Ip ACL rules written for ACL %v", aclName)
		return 0, fmt.Errorf("no rules found for ACL %v", aclName)

	}

	req := &acl_api.MacipACLAdd{
		Count: uint32(len(aclMacIPRules)),
		Tag:   []byte(aclName),
		R:     aclMacIPRules,
	}

	reply := &acl_api.MacipACLAddReply{}
	if err := vppChannel.SendRequest(req).ReceiveReply(reply); err != nil {
		return 0, fmt.Errorf("failed to write ACL %v: %v", aclName, err)
	}
	if reply.Retval != 0 {
		return 0, fmt.Errorf("error %v while writing ACL %v to VPP", reply.Retval, aclName)
	}

	log.Infof("%v Mac Ip ACL rule(s) written for ACL %v with index %v", len(aclMacIPRules), aclName, reply.ACLIndex)
	return reply.ACLIndex, nil
}

// ModifyIPAcl uses index (provided by VPP) to identify ACL which is modified.
func ModifyIPAcl(aclIndex uint32, rules []*acl.AccessLists_Acl_Rule, aclName string, log logging.Logger,
	vppChannel *api.Channel, stopwatch *measure.Stopwatch) error {
	defer func(t time.Time) {
		stopwatch.TimeLog(acl_api.ACLAddReplace{}).LogTimeEntry(time.Since(t))
	}(time.Now())

	// Prepare Ip rules
	aclIPRules, err := transformACLIpRules(rules)
	if err != nil {
		return err
	}
	if len(aclIPRules) == 0 {
		log.Debugf("No Ip ACL rules written for ACL %v", aclName)
		return nil
	}

	req := &acl_api.ACLAddReplace{
		ACLIndex: aclIndex,
		Count:    uint32(len(aclIPRules)),
		Tag:      []byte(aclName),
		R:        aclIPRules,
	}

	reply := &acl_api.ACLAddReplaceReply{}
	if err := vppChannel.SendRequest(req).ReceiveReply(reply); err != nil {
		return fmt.Errorf("failed to write ACL %v: %v", aclName, err)
	}
	if reply.Retval != 0 {
		return fmt.Errorf("error %v while writing ACL %v to VPP", reply.Retval, aclName)
	}
	log.Infof("%v Ip ACL rule(s) written for ACL %v with index %v", len(aclIPRules), aclName, aclIndex)
	return nil
}

// DeleteIPAcl removes L3/L4 ACL.
func DeleteIPAcl(aclIndex uint32, log logging.Logger, vppChannel *api.Channel, stopwatch *measure.Stopwatch) error {
	defer func(t time.Time) {
		stopwatch.TimeLog(acl_api.ACLDel{}).LogTimeEntry(time.Since(t))
	}(time.Now())

	msg := &acl_api.ACLDel{
		ACLIndex: aclIndex,
	}

	reply := &acl_api.ACLDelReply{}
	if err := vppChannel.SendRequest(msg).ReceiveReply(reply); err != nil {
		return fmt.Errorf("failed to remove L3/L4 ACL %v: %v", aclIndex, err)
	}
	if reply.Retval != 0 {
		return fmt.Errorf("error %v while removing L3/L4 ACL %v", reply.Retval, aclIndex)
	}
	log.Infof("L3/L4 ACL %v removed", aclIndex)

	return nil
}

// DeleteMacIPAcl removes L2 ACL.
func DeleteMacIPAcl(aclIndex uint32, log logging.Logger, vppChannel *api.Channel, stopwatch *measure.Stopwatch) error {
	defer func(t time.Time) {
		stopwatch.TimeLog(acl_api.MacipACLDel{}).LogTimeEntry(time.Since(t))
	}(time.Now())

	msg := &acl_api.MacipACLDel{
		ACLIndex: aclIndex,
	}

	reply := &acl_api.MacipACLDelReply{}
	if err := vppChannel.SendRequest(msg).ReceiveReply(reply); err != nil {
		return fmt.Errorf("failed to remove L2 ACL %v: %v", aclIndex, err)
	}
	if reply.Retval != 0 {
		return fmt.Errorf("error %v while removing L2 ACL %v", reply.Retval, aclIndex)
	}

	log.Infof("L2 ACL %v removed", aclIndex)
	return nil
}

// DumpInterface finds interface in VPP and returns its ACL configuration.
func DumpInterface(swIndex uint32, vppChannel *api.Channel, stopwatch *measure.Stopwatch) (*acl_api.ACLInterfaceListDetails, error) {
	defer func(t time.Time) {
		stopwatch.TimeLog(acl_api.ACLInterfaceListDump{}).LogTimeEntry(time.Since(t))
	}(time.Now())

	req := &acl_api.ACLInterfaceListDump{
		SwIfIndex: swIndex,
	}

	reply := &acl_api.ACLInterfaceListDetails{}

	if err := vppChannel.SendRequest(req).ReceiveReply(reply); err != nil {
		return nil, err
	}

	return reply, nil
}

// DumpInterfaces finds  all interfaces in VPP and returns their ACL configurations
func DumpInterfaces(vppChannel *api.Channel, stopwatch *measure.Stopwatch) ([]*acl_api.ACLInterfaceListDetails, error) {
	defer func(t time.Time) {
		stopwatch.TimeLog(acl_api.ACLInterfaceListDump{}).LogTimeEntry(time.Since(t))
	}(time.Now())

	msg := &acl_api.ACLInterfaceListDump{
		SwIfIndex: 0,
	}

	req := vppChannel.SendMultiRequest(msg)

	var aclInterfaces []*acl_api.ACLInterfaceListDetails
	for {
		reply := &acl_api.ACLInterfaceListDetails{}
		stop, err := req.ReceiveReply(reply)
		if stop {
			break
		}
		if err != nil {
			logrus.DefaultLogger().Error(err)
			return nil, err
		}
		aclInterfaces = append(aclInterfaces, reply)
	}

	return aclInterfaces, nil
}

// Method transforms provided set of IP proto ACL rules to binapi ACL rules.
func transformACLIpRules(rules []*acl.AccessLists_Acl_Rule) (aclIPRules []acl_api.ACLRule, err error) {
	for _, rule := range rules {
		aclRule := new(acl_api.ACLRule)
		// Actions
		if rule.Actions != nil {
			aclRule.IsPermit = uint8(rule.Actions.AclAction)
		}
		// Matches
		if rule.Matches != nil && rule.Matches.IpRule != nil {
			// Concerned to IP rules only
			ipRule := rule.Matches.IpRule
			// L3
			if ipRule.Ip != nil {
				aclRule, err = ipACL(ipRule.Ip, aclRule)
				if err != nil {
					return aclIPRules, err
				}
			}
			// ICMP/L4
			if ipRule.Icmp != nil {
				aclRule = icmpACL(ipRule.Icmp, aclRule)
			} else if ipRule.Tcp != nil {
				aclRule = tcpACL(ipRule.Tcp, aclRule)
			} else if ipRule.Udp != nil {
				aclRule = udpACL(ipRule.Udp, aclRule)
			} else if ipRule.Other != nil {
				aclRule = otherACL(ipRule.Other, aclRule)
			}
			aclIPRules = append(aclIPRules, *aclRule)
		}
	}
	return aclIPRules, nil
}

func transformACLMacIPRules(rules []*acl.AccessLists_Acl_Rule) (aclMacIPRules []acl_api.MacipACLRule, err error) {
	for _, rule := range rules {
		aclMacIPRule := new(acl_api.MacipACLRule)
		// Actions
		if rule.Actions != nil {
			ruleActions := rule.Actions
			aclMacIPRule.IsPermit = uint8(ruleActions.AclAction)
		}
		// Matches
		if rule.Matches != nil && rule.Matches.MacipRule != nil {
			// Concerned to MAC IP rules only
			macIPRule := rule.Matches.MacipRule
			if macIPRule == nil {
				continue
			}
			// Source IP Address + Prefix
			srcIPAddress := net.ParseIP(macIPRule.SourceAddress)
			if srcIPAddress.To4() != nil {
				aclMacIPRule.IsIpv6 = 0
				aclMacIPRule.SrcIPAddr = srcIPAddress.To4()
				aclMacIPRule.SrcIPPrefixLen = uint8(macIPRule.SourceAddressPrefix)
			} else if srcIPAddress.To16() != nil {
				aclMacIPRule.IsIpv6 = 1
				aclMacIPRule.SrcIPAddr = srcIPAddress.To16()
				aclMacIPRule.SrcIPPrefixLen = uint8(macIPRule.SourceAddressPrefix)
			} else {
				return nil, fmt.Errorf("invalid IP address %v", macIPRule.SourceAddress)
			}
			// MAC + mask
			srcMac, err := net.ParseMAC(macIPRule.SourceMacAddress)
			if err != nil {
				return aclMacIPRules, err
			}
			srcMacMask, err := net.ParseMAC(macIPRule.SourceMacAddressMask)
			if err != nil {
				return aclMacIPRules, err
			}
			aclMacIPRule.SrcMac = srcMac
			aclMacIPRule.SrcMacMask = srcMacMask
			aclMacIPRules = append(aclMacIPRules, *aclMacIPRule)
		}
	}
	return aclMacIPRules, nil
}

// The function sets an IP ACL rule fields into provided ACL Rule object. Source
// and destination addresses have to be the same IP version and contain a network mask.
func ipACL(ipRule *acl.AccessLists_Acl_Rule_Matches_IpRule_Ip, aclRule *acl_api.ACLRule) (*acl_api.ACLRule, error) {
	var (
		err        error
		srcIP      net.IP
		srcNetwork *net.IPNet
		dstIP      net.IP
		dstNetwork *net.IPNet
		srcMask    uint8
		dstMask    uint8
	)

	if strings.TrimSpace(ipRule.SourceNetwork) != "" {
		// Resolve source address
		srcIP, srcNetwork, err = net.ParseCIDR(ipRule.SourceNetwork)
		if err != nil {
			return nil, err
		}
		if srcIP.To4() == nil && srcIP.To16() == nil {
			return aclRule, fmt.Errorf("source address %v is invalid", ipRule.SourceNetwork)
		}
		maskSize, _ := srcNetwork.Mask.Size()
		srcMask = uint8(maskSize)
	}

	if strings.TrimSpace(ipRule.DestinationNetwork) != "" {
		// Resolve destination address
		dstIP, dstNetwork, err = net.ParseCIDR(ipRule.DestinationNetwork)
		if err != nil {
			return nil, err
		}
		if dstIP.To4() == nil && dstIP.To16() == nil {
			return aclRule, fmt.Errorf("destination address %v is invalid", ipRule.DestinationNetwork)
		}
		maskSize, _ := dstNetwork.Mask.Size()
		dstMask = uint8(maskSize)
	}

	// Check IP version (they should be the same), beware: IPv4 address can be converted to IPv6.
	if (srcIP.To4() != nil && dstIP.To4() == nil && dstIP.To16() != nil) ||
		(srcIP.To4() == nil && srcIP.To16() != nil && dstIP.To4() != nil) {
		return aclRule, fmt.Errorf("source address %v and destionation address %v have different IP versions",
			ipRule.SourceNetwork, ipRule.DestinationNetwork)
	}

	if srcIP.To4() != nil || dstIP.To4() != nil {
		// Ipv4 case
		aclRule.IsIpv6 = 0
		aclRule.SrcIPAddr = srcIP.To4()
		aclRule.SrcIPPrefixLen = srcMask
		aclRule.DstIPAddr = dstIP.To4()
		aclRule.DstIPPrefixLen = dstMask
	} else if srcIP.To16() != nil || dstIP.To16() != nil {
		// Ipv6 case
		aclRule.IsIpv6 = 1
		aclRule.SrcIPAddr = srcIP.To16()
		aclRule.SrcIPPrefixLen = srcMask
		aclRule.DstIPAddr = dstIP.To16()
		aclRule.DstIPPrefixLen = dstMask
	} else {
		// Both empty
		aclRule.IsIpv6 = 0
	}
	return aclRule, nil
}

// The function sets an ICMP ACL rule fields into provided ACL Rule object.
// The ranges are exclusive, use first = 0 and last = 255/65535 (icmpv4/icmpv6) to match "any".
func icmpACL(icmpRule *acl.AccessLists_Acl_Rule_Matches_IpRule_Icmp, aclRule *acl_api.ACLRule) *acl_api.ACLRule {
	if icmpRule == nil {
		return aclRule
	}
	if icmpRule.Icmpv6 {
		aclRule.Proto = ICMPv6Proto // IANA ICMPv6
		aclRule.IsIpv6 = 1
		// ICMPv6 type range
		aclRule.SrcportOrIcmptypeFirst = uint16(icmpRule.IcmpTypeRange.First)
		aclRule.SrcportOrIcmptypeLast = uint16(icmpRule.IcmpTypeRange.Last)
		// ICMPv6 code range
		aclRule.DstportOrIcmpcodeFirst = uint16(icmpRule.IcmpCodeRange.First)
		aclRule.DstportOrIcmpcodeLast = uint16(icmpRule.IcmpCodeRange.First)
	} else {
		aclRule.Proto = ICMPv4Proto // IANA ICMPv4
		aclRule.IsIpv6 = 0
		// ICMPv4 type range
		aclRule.SrcportOrIcmptypeFirst = uint16(icmpRule.IcmpTypeRange.First)
		aclRule.SrcportOrIcmptypeLast = uint16(icmpRule.IcmpTypeRange.Last)
		// ICMPv4 code range
		aclRule.DstportOrIcmpcodeFirst = uint16(icmpRule.IcmpCodeRange.First)
		aclRule.DstportOrIcmpcodeLast = uint16(icmpRule.IcmpCodeRange.Last)
	}
	return aclRule
}

// Sets an TCP ACL rule fields into provided ACL Rule object.
func tcpACL(tcpRule *acl.AccessLists_Acl_Rule_Matches_IpRule_Tcp, aclRule *acl_api.ACLRule) *acl_api.ACLRule {
	aclRule.Proto = TCPProto // IANA TCP
	aclRule.SrcportOrIcmptypeFirst = uint16(tcpRule.SourcePortRange.LowerPort)
	aclRule.SrcportOrIcmptypeLast = uint16(tcpRule.SourcePortRange.UpperPort)
	aclRule.DstportOrIcmpcodeFirst = uint16(tcpRule.DestinationPortRange.LowerPort)
	aclRule.DstportOrIcmpcodeLast = uint16(tcpRule.DestinationPortRange.UpperPort)
	aclRule.TCPFlagsValue = uint8(tcpRule.TcpFlagsValue)
	aclRule.TCPFlagsMask = uint8(tcpRule.TcpFlagsMask)
	return aclRule
}

// Sets an UDP ACL rule fields into provided ACL Rule object.
func udpACL(udpRule *acl.AccessLists_Acl_Rule_Matches_IpRule_Udp, aclRule *acl_api.ACLRule) *acl_api.ACLRule {
	aclRule.Proto = UDPProto // IANA UDP
	aclRule.SrcportOrIcmptypeFirst = uint16(udpRule.SourcePortRange.LowerPort)
	aclRule.SrcportOrIcmptypeLast = uint16(udpRule.SourcePortRange.UpperPort)
	aclRule.DstportOrIcmpcodeFirst = uint16(udpRule.DestinationPortRange.LowerPort)
	aclRule.DstportOrIcmpcodeLast = uint16(udpRule.DestinationPortRange.UpperPort)
	return aclRule
}

func otherACL(otherRule *acl.AccessLists_Acl_Rule_Matches_IpRule_Other, aclRule *acl_api.ACLRule) *acl_api.ACLRule {
	logrus.DefaultLogger().Warnf("unknown protocol: %d", otherRule.Protocol)
	return aclRule
}
