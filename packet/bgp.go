// Copyright (C) 2014 Nippon Telegraph and Telephone Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package packet

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"reflect"
	"strings"
)

// move somewhere else

const (
	AFI_IP  = 1
	AFI_IP6 = 2
)

const (
	SAFI_UNICAST                  = 1
	SAFI_MULTICAST                = 2
	SAFI_MPLS_LABEL               = 4
	SAFI_MPLS_VPN                 = 128
	SAFI_ROUTE_TARGET_CONSTRTAINS = 132
)

const (
	_ = iota
	BGP_MSG_OPEN
	BGP_MSG_UPDATE
	BGP_MSG_NOTIFICATION
	BGP_MSG_KEEPALIVE
	BGP_MSG_ROUTE_REFRESH
)

const (
	BGP_OPT_CAPABILITY = 2
)

type BGPCapabilityCode uint8

const (
	BGP_CAP_MULTIPROTOCOL          BGPCapabilityCode = 1
	BGP_CAP_ROUTE_REFRESH                            = 2
	BGP_CAP_CARRYING_LABEL_INFO                      = 4
	BGP_CAP_GRACEFUL_RESTART                         = 64
	BGP_CAP_FOUR_OCTET_AS_NUMBER                     = 65
	BGP_CAP_ENHANCED_ROUTE_REFRESH                   = 70
	BGP_CAP_ROUTE_REFRESH_CISCO                      = 128
)

func (c BGPCapabilityCode) String() string {
	switch c {
	case BGP_CAP_MULTIPROTOCOL:
		return "MultiProtocol"
	case BGP_CAP_ROUTE_REFRESH:
		return "RouteRefresh"
	case BGP_CAP_CARRYING_LABEL_INFO:
		return "CarryingLabelInfo"
	case BGP_CAP_GRACEFUL_RESTART:
		return "GracefulRestart"
	case BGP_CAP_FOUR_OCTET_AS_NUMBER:
		return "FourOctetASNumber"
	case BGP_CAP_ENHANCED_ROUTE_REFRESH:
		return "EnhancedRouteRefresh"
	case BGP_CAP_ROUTE_REFRESH_CISCO:
		return "RouteRefreshCisco"
	}
	return "Unknown"
}

func structName(m interface{}) string {
	r := reflect.TypeOf(m)
	return strings.TrimLeft(r.String(), "*bgp.")
}

type ParameterCapabilityInterface interface {
	DecodeFromBytes([]byte) error
	Encode() ([]byte, error)
	Len() int
}

type DefaultParameterCapability struct {
	CapCode  BGPCapabilityCode
	CapLen   uint8
	CapValue []byte
}

func (c *DefaultParameterCapability) DecodeFromBytes(data []byte) error {
	c.CapCode = BGPCapabilityCode(data[0])
	c.CapLen = data[1]
	if uint8(len(data)) < 2+c.CapLen {
		return fmt.Errorf("Not all OptionParameterCapability bytes available")
	}
	c.CapValue = data[2 : 2+c.CapLen]
	return nil
}

func (c *DefaultParameterCapability) Len() int {
	return int(c.CapLen + 2)
}

type CapMultiProtocolValue struct {
	AFI  uint16
	SAFI uint8
}

type CapMultiProtocol struct {
	DefaultParameterCapability
	CapValue CapMultiProtocolValue
}

func (c *CapMultiProtocol) DecodeFromBytes(data []byte) error {
	c.DefaultParameterCapability.DecodeFromBytes(data)
	data = data[2:]
	if len(data) < 4 {
		return fmt.Errorf("Not all CapabilityMultiProtocol bytes available")
	}
	c.CapValue.AFI = binary.BigEndian.Uint16(data[0:2])
	c.CapValue.SAFI = data[3]
	return nil
}

type CapRouteRefresh struct {
	DefaultParameterCapability
}

type CapCarryingLabelInfo struct {
	DefaultParameterCapability
}

type CapGracefulRestartTuples struct {
	AFI   uint16
	SAFI  uint8
	Flags uint8
}

type CapGracefulRestartValue struct {
	Flags  uint8
	Time   uint16
	Tuples []CapGracefulRestartTuples
}

type CapGracefulRestart struct {
	DefaultParameterCapability
	CapValue CapGracefulRestartValue
}

func (c *CapGracefulRestart) DecodeFromBytes(data []byte) error {
	c.DefaultParameterCapability.DecodeFromBytes(data)
	data = data[2:]
	restart := binary.BigEndian.Uint16(data[0:2])
	c.CapValue.Flags = uint8(restart >> 12)
	c.CapValue.Time = restart & 0xfff
	data = data[2:]
	for len(data) >= 4 {
		t := CapGracefulRestartTuples{binary.BigEndian.Uint16(data[0:2]),
			data[2], data[3]}
		c.CapValue.Tuples = append(c.CapValue.Tuples, t)
		data = data[4:]
	}
	return nil
}

type CapFourOctetASNumber struct {
	DefaultParameterCapability
	CapValue uint32
}

func (c *CapFourOctetASNumber) DecodeFromBytes(data []byte) error {
	c.DefaultParameterCapability.DecodeFromBytes(data)
	data = data[2:]
	if len(data) < 4 {
		return fmt.Errorf("Not all CapabilityMultiProtocol bytes available")
	}
	c.CapValue = binary.BigEndian.Uint32(data[0:4])
	return nil
}

type CapEnhancedRouteRefresh struct {
	DefaultParameterCapability
}

type CapRouteRefreshCisco struct {
	DefaultParameterCapability
}

type CapUnknown struct {
	DefaultParameterCapability
}

type OptionParameterInterface interface {
}

type OptionParameterCapability struct {
	ParamType  uint8
	ParamLen   uint8
	Capability []ParameterCapabilityInterface
}

type OptionParameterUnknown struct {
	ParamType uint8
	ParamLen  uint8
	Value     []byte
}

func (o *OptionParameterCapability) DecodeFromBytes(data []byte) error {
	if uint8(len(data)) < o.ParamLen {
		return fmt.Errorf("Not all OptionParameterCapability bytes available")
	}
	for len(data) >= 2 {
		var c ParameterCapabilityInterface
		switch BGPCapabilityCode(data[0]) {
		case BGP_CAP_MULTIPROTOCOL:
			c = &CapMultiProtocol{}
		case BGP_CAP_ROUTE_REFRESH:
			c = &CapRouteRefresh{}
		case BGP_CAP_CARRYING_LABEL_INFO:
			c = &CapCarryingLabelInfo{}
		case BGP_CAP_GRACEFUL_RESTART:
			c = &CapGracefulRestart{}
		case BGP_CAP_FOUR_OCTET_AS_NUMBER:
			c = &CapFourOctetASNumber{}
		case BGP_CAP_ENHANCED_ROUTE_REFRESH:
			c = &CapEnhancedRouteRefresh{}
		case BGP_CAP_ROUTE_REFRESH_CISCO:
			c = &CapRouteRefreshCisco{}
		default:
			c = &CapUnknown{}
		}
		err := c.DecodeFromBytes(data)
		if err != nil {
			return nil
		}
		o.Capability = append(o.Capability, c)
		data = data[c.Len():]
	}
	return nil
}

type BGPOpen struct {
	Version     uint8
	MyAS        uint16
	HoldTime    uint16
	ID          net.IP
	OptParamLen uint8
	OptParams   []OptionParameterInterface
}

func (msg *BGPOpen) DecodeFromBytes(data []byte) error {
	msg.Version = data[0]
	msg.MyAS = binary.BigEndian.Uint16(data[1:3])
	msg.HoldTime = binary.BigEndian.Uint16(data[3:5])
	msg.ID = data[5:9]
	msg.OptParamLen = data[9]
	data = data[10:]
	if uint8(len(data)) < msg.OptParamLen {
		return fmt.Errorf("Not all BGP Open message bytes available")
	}

	for rest := msg.OptParamLen; rest > 0; {
		paramtype := data[0]
		paramlen := data[1]
		rest -= paramlen + 2

		if paramtype == BGP_OPT_CAPABILITY {
			p := OptionParameterCapability{}
			p.ParamType = paramtype
			p.ParamLen = paramlen
			p.DecodeFromBytes(data[2 : 2+paramlen])
			msg.OptParams = append(msg.OptParams, p)
		} else {
			p := OptionParameterUnknown{}
			p.ParamType = paramtype
			p.ParamLen = paramlen
			p.Value = data[2 : 2+paramlen]
			msg.OptParams = append(msg.OptParams, p)
		}
		data = data[2+paramlen:]
	}
	return nil
}

func (msg *BGPOpen) Encode() ([]byte, error) {

	buf := make([]byte, 10)
	buf[0] = byte(msg.Version)
	binary.BigEndian.PutUint16(buf[1:3], msg.MyAS)
	binary.BigEndian.PutUint16(buf[3:5], msg.HoldTime)
	buf[5:9] = msg.ID
	buf[9] = msg.OptParamLen

	if msg.OptParamLen > 0 {

		if msg.OptParamLen != uint8(len(msg.OptParams)) {
			return nil, fmt.Errorf("Option parameter length mismatch")
		}

		for i, p := range msg.OptParams {
			optBuf := make([]byte, 2 + p.ParamLen)
			optBuf[0] = p.ParamType
			optBuf[1] = p.ParamLen
			optBuf[2:2 + p.ParamLen] = p.Value
			buf = append(buf, optBuf)
		}
	}

	return buf, nil
}

type AddrPrefixInterface interface {
	DecodeFromBytes([]byte) error
	Len() int
}

type IPAddrPrefixDefault struct {
	Length uint8
	Prefix net.IP
}

func (r *IPAddrPrefixDefault) decodePrefix(data []byte, bitlen uint8, addrlen uint8) error {
	bytelen := (bitlen + 7) / 8
	b := make([]byte, addrlen)
	copy(b, data[:bytelen])
	r.Prefix = b
	return nil
}

func (r *IPAddrPrefixDefault) Len() int {
	return int(1 + ((r.Length + 7) / 8))
}

type IPAddrPrefix struct {
	IPAddrPrefixDefault
	addrlen uint8
}

func (r *IPAddrPrefix) DecodeFromBytes(data []byte) error {
	r.Length = data[0]
	if r.addrlen == 0 {
		r.addrlen = 4
	}
	r.decodePrefix(data[1:], r.Length, r.addrlen)
	return nil
}

type IPv6AddrPrefix struct {
	IPAddrPrefix
}

func NewIPv6AddrPrefix() *IPv6AddrPrefix {
	p := &IPv6AddrPrefix{}
	p.addrlen = 16
	return p
}

type WithdrawnRoute struct {
	IPAddrPrefix
}

const (
	BGP_RD_TWO_OCTET_AS = iota
	BGP_RD_IPV4_ADDRESS
	BGP_RD_FOUR_OCTET_AS
)

type RouteDistinguisherInterface interface {
	DecodeFromBytes([]byte) error
	Len() int
}

type DefaultRouteDistinguisher struct {
	Type  uint16
	Value []byte
}

func (rd DefaultRouteDistinguisher) DecodeFromBytes(data []byte) error {
	rd.Type = binary.BigEndian.Uint16(data[0:2])
	rd.Value = data[2:8]
	return nil
}

func (rd *DefaultRouteDistinguisher) Len() int { return 8 }

type RouteDistinguisherTwoOctetASValue struct {
	Admin    uint16
	Assigned uint32
}

type RouteDistinguisherTwoOctetAS struct {
	DefaultRouteDistinguisher
	Value RouteDistinguisherTwoOctetASValue
}

type RouteDistinguisherIPAddressASValue struct {
	Admin    net.IP
	Assigned uint16
}

type RouteDistinguisherIPAddressAS struct {
	DefaultRouteDistinguisher
	Value RouteDistinguisherIPAddressASValue
}

type RouteDistinguisherFourOctetASValue struct {
	Admin    uint32
	Assigned uint16
}

type RouteDistinguisherFourOctetAS struct {
	DefaultRouteDistinguisher
	Value RouteDistinguisherFourOctetASValue
}

type RouteDistinguisherUnknown struct {
	DefaultRouteDistinguisher
}

func getRouteDistinguisher(data []byte) RouteDistinguisherInterface {
	switch binary.BigEndian.Uint16(data[0:2]) {
	case BGP_RD_TWO_OCTET_AS:
		rd := &RouteDistinguisherTwoOctetAS{}
		rd.Value.Admin = binary.BigEndian.Uint16(data[2:4])
		rd.Value.Assigned = binary.BigEndian.Uint32(data[4:8])
		return rd
	case BGP_RD_IPV4_ADDRESS:
		rd := &RouteDistinguisherIPAddressAS{}
		rd.Value.Admin = data[2:6]
		rd.Value.Assigned = binary.BigEndian.Uint16(data[6:8])
		return rd
	case BGP_RD_FOUR_OCTET_AS:
		rd := &RouteDistinguisherFourOctetAS{}
		rd.Value.Admin = binary.BigEndian.Uint32(data[2:6])
		rd.Value.Assigned = binary.BigEndian.Uint16(data[6:8])
		return rd
	}

	return &RouteDistinguisherUnknown{}
}

type Label struct {
	Labels []uint32
}

func (l *Label) DecodeFromBytes(data []byte) error {
	labels := []uint32{}
	foundBottom := false
	for len(data) >= 4 {
		label := uint32(data[0]<<16 | data[1]<<8 | data[2])
		data = data[3:]
		labels = append(labels, label>>4)
		if label&1 == 1 {
			foundBottom = true
			break
		}
	}
	if foundBottom == false {
		l.Labels = []uint32{}
		return nil
	}
	l.Labels = labels
	return nil
}

func (l *Label) Len() int { return 3 * len(l.Labels) }

type LabelledVPNIPAddrPrefix struct {
	IPAddrPrefixDefault
	Labels  Label
	RD      RouteDistinguisherInterface
	addrlen uint8
}

func (l *LabelledVPNIPAddrPrefix) DecodeFromBytes(data []byte) error {
	l.Length = uint8(data[0])
	data = data[1:]
	l.Labels.DecodeFromBytes(data)
	if int(l.Length)-8*(l.Labels.Len()) < 0 {
		l.Labels.Labels = []uint32{}
	}
	data = data[l.Labels.Len():]
	l.RD = getRouteDistinguisher(data)
	data = data[l.RD.Len():]
	restbits := int(l.Length) - 8*(l.Labels.Len()+l.RD.Len())
	l.decodePrefix(data, uint8(restbits), l.addrlen)
	return nil
}

func NewLabelledVPNIPAddrPrefix() *LabelledVPNIPAddrPrefix {
	p := &LabelledVPNIPAddrPrefix{}
	p.addrlen = 4
	return p
}

type LabelledVPNIPv6AddrPrefix struct {
	LabelledVPNIPAddrPrefix
}

func NewLabelledVPNIPv6AddrPrefix() *LabelledVPNIPv6AddrPrefix {
	p := &LabelledVPNIPv6AddrPrefix{}
	p.addrlen = 16
	return p
}

type LabelledIPAddrPrefix struct {
	IPAddrPrefixDefault
	Labels  Label
	addrlen uint8
}

func (r *IPAddrPrefix) decodeNextHop(data []byte) net.IP {
	if r.addrlen == 0 {
		r.addrlen = 4
	}
	var next net.IP = data[0:r.addrlen]
	return next
}

func (r *LabelledVPNIPAddrPrefix) decodeNextHop(data []byte) net.IP {
	// skip rd
	var next net.IP = data[8 : 8+r.addrlen]
	return next
}

func (r *LabelledIPAddrPrefix) decodeNextHop(data []byte) net.IP {
	var next net.IP = data[0:r.addrlen]
	return next
}

func (l *LabelledIPAddrPrefix) DecodeFromBytes(data []byte) error {
	l.Length = uint8(data[0])
	data = data[1:]
	l.Labels.DecodeFromBytes(data)
	if int(l.Length)-8*(l.Labels.Len()) < 0 {
		l.Labels.Labels = []uint32{}
	}
	restbits := int(l.Length) - 8*(l.Labels.Len())
	data = data[l.Labels.Len():]
	l.decodePrefix(data, uint8(restbits), l.addrlen)
	return nil
}

func NewLabelledIPAddrPrefix() *LabelledIPAddrPrefix {
	p := &LabelledIPAddrPrefix{}
	p.addrlen = 4
	return p
}

type LabelledIPv6AddrPrefix struct {
	LabelledIPAddrPrefix
}

func NewLabelledIPv6AddrPrefix() *LabelledIPv6AddrPrefix {
	p := &LabelledIPv6AddrPrefix{}
	p.addrlen = 16
	return p
}

type RouteTargetMembershipNLRI struct {
	AS          uint32
	RouteTarget ExtendedCommunityInterface
}

func (n *RouteTargetMembershipNLRI) DecodeFromBytes(data []byte) error {
	n.AS = binary.BigEndian.Uint32(data[0:4])
	n.RouteTarget = parseExtended(data[4:])
	return nil
}

func (n *RouteTargetMembershipNLRI) Len() int { return 12 }

func rfshift(afi uint16, safi uint8) int {
	return int(afi)<<16 | int(safi)
}

const (
	RF_IPv4_UC   = AFI_IP<<16 | SAFI_UNICAST
	RF_IPv6_UC   = AFI_IP6<<16 | SAFI_UNICAST
	RF_IPv4_VPN  = AFI_IP<<16 | SAFI_MPLS_VPN
	RF_IPv6_VPN  = AFI_IP6<<16 | SAFI_MPLS_VPN
	RF_IPv4_MPLS = AFI_IP<<16 | SAFI_MPLS_LABEL
	RF_IPv6_MPLS = AFI_IP6<<16 | SAFI_MPLS_LABEL
	RF_RTC_UC    = AFI_IP<<16 | SAFI_ROUTE_TARGET_CONSTRTAINS
)

func routeFamilyPrefix(afi uint16, safi uint8) (prefix AddrPrefixInterface) {
	switch rfshift(afi, safi) {
	case RF_IPv4_UC:
		prefix = &IPAddrPrefix{}
	case RF_IPv6_UC:
		prefix = NewIPv6AddrPrefix()
	case RF_IPv4_VPN:
		prefix = NewLabelledVPNIPAddrPrefix()
	case RF_IPv6_VPN:
		prefix = NewLabelledVPNIPv6AddrPrefix()
	case RF_IPv4_MPLS:
		prefix = NewLabelledIPAddrPrefix()
	case RF_IPv6_MPLS:
		prefix = NewLabelledIPv6AddrPrefix()
	case RF_RTC_UC:
		prefix = &RouteTargetMembershipNLRI{}
	}
	return prefix
}

const (
	BGP_ATTR_FLAG_EXTENDED_LENGTH = 1 << 4
	BGP_ATTR_FLAG_PARTIAL         = 1 << 5
	BGP_ATTR_FLAG_TRANSITIVE      = 1 << 6
	BGP_ATTR_FLAG_OPTIONAL        = 1 << 7
)

const (
	_ = iota
	BGP_ATTR_TYPE_ORIGIN
	BGP_ATTR_TYPE_AS_PATH
	BGP_ATTR_TYPE_NEXT_HOP
	BGP_ATTR_TYPE_MULTI_EXIT_DISC
	BGP_ATTR_TYPE_LOCAL_PREF
	BGP_ATTR_TYPE_ATOMIC_AGGREGATE
	BGP_ATTR_TYPE_AGGREGATOR
	BGP_ATTR_TYPE_COMMUNITIES
	BGP_ATTR_TYPE_ORIGINATOR_ID
	BGP_ATTR_TYPE_CLUSTER_LIST
	_
	_
	_
	BGP_ATTR_TYPE_MP_REACH_NLRI
	BGP_ATTR_TYPE_MP_UNREACH_NLRI
	BGP_ATTR_TYPE_EXTENDED_COMMUNITIES
	BGP_ATTR_TYPE_AS4_PATH
	BGP_ATTR_TYPE_AS4_AGGREGATOR
)

// NOTIFICATION Error Code  RFC 4271 4.5.
const (
	_ = iota
	BGP_ERROR_MESSAGE_HEADER_ERROR
	BGP_ERROR_OPEN_MESSAGE_ERROR
	BGP_ERROR_UPDATE_MESSAGE_ERROR
	BGP_ERROR_HOLD_TIMER_EXPIRED
	BGP_ERROR_FSM_ERROR
	BGP_ERROR_CEASE
)

// NOTIFICATION Error Subcode for BGP_ERROR_MESSAGE_HEADER_ERROR
const (
	_ = iota
	BGP_ERROR_SUB_CONNECTION_NOT_SYNCHRONIZED
	BGP_ERROR_SUB_BAD_MESSAGE_LENGTH
	BGP_ERROR_SUB_BAD_MESSAGE_TYPE
)

// NOTIFICATION Error Subcode for BGP_ERROR_OPEN_MESSAGE_ERROR
const (
	_ = iota
	BGP_ERROR_SUB_UNSUPPORTED_VERSION_NUMBER
	BGP_ERROR_SUB_BAD_PEER_AS
	BGP_ERROR_SUB_BAD_BGP_IDENTIFIER
	BGP_ERROR_SUB_UNSUPPORTED_OPTIONAL_PARAMETER
	BGP_ERROR_SUB_AUTHENTICATION_FAILURE
	BGP_ERROR_SUB_UNACCEPTABLE_HOLD_TIME
)

// NOTIFICATION Error Subcode for BGP_ERROR_UPDATE_MESSAGE_ERROR
const (
	_ = iota
	BGP_ERROR_SUB_MALFORMED_ATTRIBUTE_LIST
	BGP_ERROR_SUB_UNRECOGNIZED_WELL_KNOWN_ATTRIBUTE
	BGP_ERROR_SUB_MISSING_WELL_KNOWN_ATTRIBUTE
	BGP_ERROR_SUB_ATTRIBUTE_FLAGS_ERROR
	BGP_ERROR_SUB_ATTRIBUTE_LENGTH_ERROR
	BGP_ERROR_SUB_INVALID_ORIGIN_ATTRIBUTE
	BGP_ERROR_SUB_ROUTING_LOOP
	BGP_ERROR_SUB_INVALID_NEXT_HOP_ATTRIBUTE
	BGP_ERROR_SUB_OPTIONAL_ATTRIBUTE_ERROR
	BGP_ERROR_SUB_INVALID_NETWORK_FIELD
	BGP_ERROR_SUB_MALFORMED_AS_PATH
)

// NOTIFICATION Error Subcode for BGP_ERROR_HOLD_TIMER_EXPIRED
const (
	_ = iota
	BGP_ERROR_SUB_HOLD_TIMER_EXPIRED
)

// NOTIFICATION Error Subcode for BGP_ERROR_FSM_ERROR
const (
	_ = iota
	BGP_ERROR_SUB_FSM_ERROR
)

// NOTIFICATION Error Subcode for BGP_ERROR_CEASE  (RFC 4486)
const (
	_ = iota
	BGP_ERROR_SUB_MAXIMUM_NUMBER_OF_PREFIXES_REACHED
	BGP_ERROR_SUB_ADMINISTRATIVE_SHUTDOWN
	BGP_ERROR_SUB_PEER_DECONFIGURED
	BGP_ERROR_SUB_ADMINISTRATIVE_RESET
	BGP_ERROR_SUB_CONNECTION_RESET
	BGP_ERROR_SUB_OTHER_CONFIGURATION_CHANGE
	BGP_ERROR_SUB_CONNECTION_COLLISION_RESOLUTION
	BGP_ERROR_SUB_OUT_OF_RESOURCES
)

type PathAttributeInterface interface {
	DecodeFromBytes([]byte) error
	Len() int
}

type PathAttribute struct {
	Flags  uint8
	Type   uint8
	Length uint16
	Value  []byte
}

func (p *PathAttribute) Len() int {
	l := 2 + p.Length
	if p.Flags&BGP_ATTR_FLAG_EXTENDED_LENGTH != 0 {
		l += 2
	} else {
		l += 1
	}
	return int(l)
}

func (p *PathAttribute) DecodeFromBytes(data []byte) error {
	p.Flags = data[0]
	p.Type = data[1]

	if p.Flags&BGP_ATTR_FLAG_EXTENDED_LENGTH != 0 {
		p.Length = binary.BigEndian.Uint16(data[2:4])
		data = data[4:]
	} else {
		p.Length = uint16(data[2])
		data = data[3:]
	}
	p.Value = data[:p.Length]

	return nil
}

type PathAttributeOrigin struct {
	PathAttribute
}

type AsPathParam struct {
	Type uint8
	Num  uint8
	AS   []uint32
}

type DefaultAsPath struct {
}

func (p *DefaultAsPath) isValidAspath(data []byte) bool {
	for len(data) > 0 {
		segType := data[0]
		asNum := data[1]
		if segType == 0 || segType > 4 {
			return false
		}
		data = data[2:]
		if len(data) < int(asNum)*2 {
			return false
		}
		data = data[2*asNum:]
	}
	return true
}

// TODO: could marge two functions nicely with reflect?
func (p *DefaultAsPath) decodeAspath(data []byte) []AsPathParam {
	var param []AsPathParam

	for len(data) > 0 {
		a := AsPathParam{}
		a.Type = data[0]
		a.Num = data[1]
		data = data[2:]
		for i := 0; i < int(a.Num); i++ {
			a.AS = append(a.AS, uint32(binary.BigEndian.Uint16(data)))
			data = data[2:]
		}
		param = append(param, a)
	}
	return param
}

func (p *DefaultAsPath) decodeAs4path(data []byte) []AsPathParam {
	var param []AsPathParam

	for len(data) > 0 {
		a := AsPathParam{}
		a.Type = data[0]
		a.Num = data[1]
		data = data[2:]
		for i := 0; i < int(a.Num); i++ {
			a.AS = append(a.AS, binary.BigEndian.Uint32(data))
			data = data[4:]
		}
		param = append(param, a)
	}
	return param
}

type PathAttributeAsPath struct {
	DefaultAsPath
	PathAttribute
	Value []AsPathParam
}

func (p *PathAttributeAsPath) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)
	if p.DefaultAsPath.isValidAspath(p.PathAttribute.Value) {
		p.Value = p.DefaultAsPath.decodeAspath(p.PathAttribute.Value)
	} else {
		p.Value = p.DefaultAsPath.decodeAs4path(p.PathAttribute.Value)
	}
	return nil
}

type PathAttributeNextHop struct {
	PathAttribute
	Value net.IP
}

func (p *PathAttributeNextHop) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)
	p.Value = p.PathAttribute.Value
	return nil
}

type PathAttributeMultiExitDisc struct {
	PathAttribute
	Value uint32
}

func (p *PathAttributeMultiExitDisc) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)
	p.Value = binary.BigEndian.Uint32(p.PathAttribute.Value)
	return nil
}

type PathAttributeLocalPref struct {
	PathAttribute
	Value uint32
}

func (p *PathAttributeLocalPref) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)
	p.Value = binary.BigEndian.Uint32(p.PathAttribute.Value)
	return nil
}

type PathAttributeAtomicAggregate struct {
	PathAttribute
}

type PathAttributeAggregatorParam struct {
	AS      uint32
	Address net.IP
}

type PathAttributeAggregator struct {
	PathAttribute
	Value PathAttributeAggregatorParam
}

func (p *PathAttributeAggregator) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)
	if len(p.PathAttribute.Value) == 6 {
		p.Value.AS = uint32(binary.BigEndian.Uint16(p.PathAttribute.Value[0:2]))
		p.Value.Address = p.PathAttribute.Value[2:]
	} else {
		p.Value.AS = binary.BigEndian.Uint32(p.PathAttribute.Value[0:4])
		p.Value.Address = p.PathAttribute.Value[4:]
	}
	return nil
}

type PathAttributeCommunities struct {
	PathAttribute
	Value []uint32
}

func (p *PathAttributeCommunities) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)
	value := p.PathAttribute.Value
	for len(value) >= 4 {
		p.Value = append(p.Value, binary.BigEndian.Uint32(value))
		value = value[4:]
	}
	return nil
}

type PathAttributeOriginatorId struct {
	PathAttribute
	Value net.IP
}

func (p *PathAttributeOriginatorId) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)
	p.Value = p.PathAttribute.Value
	return nil
}

type PathAttributeClusterList struct {
	PathAttribute
	Value []net.IP
}

func (p *PathAttributeClusterList) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)
	value := p.PathAttribute.Value
	for len(value) >= 4 {
		p.Value = append(p.Value, value[:4])
		value = value[4:]
	}
	return nil
}

type PathAttributeMpReachNLRI struct {
	PathAttribute
	Nexthop net.IP
	Value   []AddrPrefixInterface
}

func (p *PathAttributeMpReachNLRI) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)

	value := p.PathAttribute.Value
	afi := binary.BigEndian.Uint16(value[0:2])
	safi := value[2]
	nexthopLen := value[3]
	nexthopbin := value[4 : 4+nexthopLen]
	value = value[4+nexthopLen:]
	if nexthopLen > 0 {
		offset := 0
		if safi == SAFI_MPLS_VPN {
			offset = 8
		}
		addrlen := 4
		if afi == AFI_IP6 {
			addrlen = 16
		}
		p.Nexthop = nexthopbin[offset : +offset+addrlen]
	}
	// skip reserved
	value = value[1:]
	for len(value) > 0 {
		prefix := routeFamilyPrefix(afi, safi)
		prefix.DecodeFromBytes(value)
		value = value[prefix.Len():]
		p.Value = append(p.Value, prefix)
	}
	return nil
}

type PathAttributeMpUnreachNLRI struct {
	PathAttribute
	Value []AddrPrefixInterface
}

func (p *PathAttributeMpUnreachNLRI) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)

	value := p.PathAttribute.Value
	afi := binary.BigEndian.Uint16(value[0:2])
	safi := value[2]
	addrprefix := routeFamilyPrefix(afi, safi)
	value = value[3:]
	for len(value) > 0 {
		prefix := addrprefix
		prefix.DecodeFromBytes(value)
		value = value[prefix.Len():]
		p.Value = append(p.Value, prefix)
	}
	return nil
}

type ExtendedCommunityInterface interface {
}

type TwoOctetAsSpecificExtended struct {
	AS         uint16
	LocalAdmin uint32
}

type IPv4AddressSpecificExtended struct {
	SubType    uint8
	IPv4       net.IP
	LocalAdmin uint16
}

type FourOctetAsSpecificExtended struct {
	AS         net.IP
	LocalAdmin uint16
}

type OpaqueExtended struct {
	Value []byte
}

type UnknownExtended struct {
	Value []byte
}

type PathAttributeExtendedCommunities struct {
	PathAttribute
	Value []ExtendedCommunityInterface
}

func parseExtended(data []byte) ExtendedCommunityInterface {
	typehigh := data[0] & ^uint8(0x40)
	switch typehigh {
	case 0:
		e := &TwoOctetAsSpecificExtended{}
		e.AS = binary.BigEndian.Uint16(data[2:4])
		e.LocalAdmin = binary.BigEndian.Uint32(data[4:8])
		return e
	case 1:
		e := &IPv4AddressSpecificExtended{}
		e.SubType = data[1]
		e.IPv4 = data[2:6]
		e.LocalAdmin = binary.BigEndian.Uint16(data[6:8])
		return e
	case 2:
		e := &FourOctetAsSpecificExtended{}
		e.AS = data[2:6]
		e.LocalAdmin = binary.BigEndian.Uint16(data[6:8])
		return e
	case 3:
		e := &OpaqueExtended{}
		e.Value = data[2:8]
		return e
	}
	e := &UnknownExtended{}
	e.Value = data[2:8]
	return e
}

func (p *PathAttributeExtendedCommunities) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)

	value := p.PathAttribute.Value
	for len(value) >= 8 {
		e := parseExtended(value)
		p.Value = append(p.Value, e)
		value = value[8:]
	}
	return nil
}

type PathAttributeAs4Path struct {
	PathAttribute
	Value []AsPathParam
	DefaultAsPath
}

func (p *PathAttributeAs4Path) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)
	p.Value = p.DefaultAsPath.decodeAs4path(p.PathAttribute.Value)
	return nil
}

type PathAttributeAs4Aggregator struct {
	PathAttribute
	Value PathAttributeAggregatorParam
}

func (p *PathAttributeAs4Aggregator) DecodeFromBytes(data []byte) error {
	p.PathAttribute.DecodeFromBytes(data)

	p.Value.AS = binary.BigEndian.Uint32(p.PathAttribute.Value[0:4])
	p.Value.Address = p.PathAttribute.Value[4:]
	return nil
}

type PathAttributeUnknown struct {
	PathAttribute
}

func getPathAttribute(data []byte) PathAttributeInterface {
	switch data[1] {
	case BGP_ATTR_TYPE_ORIGIN:
		return &PathAttributeOrigin{}
	case BGP_ATTR_TYPE_AS_PATH:
		return &PathAttributeAsPath{}
	case BGP_ATTR_TYPE_NEXT_HOP:
		return &PathAttributeNextHop{}
	case BGP_ATTR_TYPE_MULTI_EXIT_DISC:
		return &PathAttributeMultiExitDisc{}
	case BGP_ATTR_TYPE_LOCAL_PREF:
		return &PathAttributeLocalPref{}
	case BGP_ATTR_TYPE_ATOMIC_AGGREGATE:
		return &PathAttributeAtomicAggregate{}
	case BGP_ATTR_TYPE_AGGREGATOR:
		return &PathAttributeAggregator{}
	case BGP_ATTR_TYPE_COMMUNITIES:
		return &PathAttributeCommunities{}
	case BGP_ATTR_TYPE_ORIGINATOR_ID:
		return &PathAttributeOriginatorId{}
	case BGP_ATTR_TYPE_CLUSTER_LIST:
		return &PathAttributeClusterList{}
	case BGP_ATTR_TYPE_MP_REACH_NLRI:
		return &PathAttributeMpReachNLRI{}
	case BGP_ATTR_TYPE_MP_UNREACH_NLRI:
		return &PathAttributeMpUnreachNLRI{}
	case BGP_ATTR_TYPE_EXTENDED_COMMUNITIES:
		return &PathAttributeExtendedCommunities{}
	case BGP_ATTR_TYPE_AS4_PATH:
		return &PathAttributeAs4Path{}
	case BGP_ATTR_TYPE_AS4_AGGREGATOR:
		return &PathAttributeAs4Aggregator{}
	}
	return &PathAttributeUnknown{}
}

type NLRInfo struct {
	IPAddrPrefix
}

type BGPUpdate struct {
	WithdrawnRoutesLen    uint16
	WithdrawnRoutes       []WithdrawnRoute
	TotalPathAttributeLen uint16
	PathAttributes        []PathAttributeInterface
	NLRI                  []NLRInfo
}

func (msg *BGPUpdate) DecodeFromBytes(data []byte) error {
	msg.WithdrawnRoutesLen = binary.BigEndian.Uint16(data[0:2])
	data = data[2:]
	for routelen := msg.WithdrawnRoutesLen; routelen > 0; {
		w := WithdrawnRoute{}
		w.DecodeFromBytes(data)
		routelen -= uint16(w.Len())
		data = data[w.Len():]
		msg.WithdrawnRoutes = append(msg.WithdrawnRoutes, w)
	}
	msg.TotalPathAttributeLen = binary.BigEndian.Uint16(data[0:2])
	data = data[2:]
	for pathlen := msg.TotalPathAttributeLen; pathlen > 0; {
		p := getPathAttribute(data)
		p.DecodeFromBytes(data)
		pathlen -= uint16(p.Len())
		data = data[p.Len():]
		msg.PathAttributes = append(msg.PathAttributes, p)
	}

	for restlen := len(data); restlen > 0; {
		n := NLRInfo{}
		n.DecodeFromBytes(data)
		restlen -= n.Len()
		data = data[n.Len():]
		msg.NLRI = append(msg.NLRI, n)
	}

	return nil
}

type BGPNotification struct {
	ErrorCode    uint8
	ErrorSubcode uint8
	Data         []byte
}

func (msg *BGPNotification) DecodeFromBytes(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("Not all Notificaiton bytes available")
	}
	msg.ErrorCode = data[0]
	msg.ErrorSubcode = data[1]
	if len(data) > 2 {
		msg.Data = data[2:]
	}
	return nil
}

type BGPKeepAlive struct {
}

func (msg *BGPKeepAlive) DecodeFromBytes(data []byte) error {
	return nil
}

type BGPRouteRefresh struct {
	AFI         uint16
	Demarcation uint8
	SAFI        uint8
}

func (msg *BGPRouteRefresh) DecodeFromBytes(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("Not all RouteRefresh bytes available")
	}
	msg.AFI = binary.BigEndian.Uint16(data[0:2])
	msg.Demarcation = data[2]
	msg.SAFI = data[3]
	return nil
}

type BGPBody interface {
	DecodeFromBytes([]byte) error
}

type BGPHeader struct {
	Marker []byte
	Len    uint16
	Type   uint8
}

func (msg *BGPHeader) DecodeFromBytes(data []byte) error {
	// minimum BGP message length
	if uint16(len(data)) < 19 {
		return fmt.Errorf("Not all BGP message header")
	}
	msg.Len = binary.BigEndian.Uint16(data[16:18])
	msg.Type = data[18]
	if len(data) < int(msg.Len) {
		return fmt.Errorf("Not all BGP message bytes available")
	}
	return nil
}

func (msg *BGPHeader) Encode() ([]byte, error) {

	data := make([]byte, 19)
	for i, _ := range data[0:16] {
		data[i] = uint8(1)
	}

	binary.BigEndian.PutUint16(data[16:18], msg.Len)
	data[18] = msg.Type
	return data, nil
}

type BGPMessage struct {
	Header BGPHeader
	Body   BGPBody
}

func ParseBGPMessage(data []byte) (*BGPMessage, error) {
	msg := &BGPMessage{}
	err := msg.Header.DecodeFromBytes(data)
	if err != nil {
		return nil, err
	}
	data = data[19:msg.Header.Len]
	switch msg.Header.Type {
	case BGP_MSG_OPEN:
		msg.Body = &BGPOpen{}
	case BGP_MSG_UPDATE:
		msg.Body = &BGPUpdate{}
	case BGP_MSG_NOTIFICATION:
		msg.Body = &BGPNotification{}
	case BGP_MSG_KEEPALIVE:
		msg.Body = &BGPKeepAlive{}
	case BGP_MSG_ROUTE_REFRESH:
		msg.Body = &BGPRouteRefresh{}
	}
	err = msg.Body.DecodeFromBytes(data)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func SerializeBGPMessage(msg *BGPMessage) ([]byte, error){

	header, err:= msg.Header.Encode()
	body , err := msg.Body.Encode()
	data := make([]byte, len(header) + len(body))
	append(data, header, body)
	return data, nil

}

// BMP

type BMPHeader struct {
	Version uint8
	Length  uint32
	Type    uint8
}

const (
	BMP_HEADER_SIZE = 6
)

func (msg *BMPHeader) DecodeFromBytes(data []byte) error {
	msg.Version = data[0]
	if data[0] != 3 {
		return fmt.Errorf("error version")
	}
	msg.Length = binary.BigEndian.Uint32(data[1:5])
	msg.Type = data[5]
	return nil
}

func (msg *BMPHeader) Len() int {
	return int(msg.Length)
}

type BMPPeerHeader struct {
	PeerType          uint8
	IsPostPolicy      bool
	PeerDistinguisher uint64
	PeerAddress       net.IP
	PeerAS            uint32
	PeerBGPID         net.IP
	Timestamp         float64
	flags             uint8
}

func (msg *BMPPeerHeader) DecodeFromBytes(data []byte) error {
	data = data[6:]

	msg.PeerType = data[0]
	flags := data[1]
	msg.flags = flags
	if flags&1<<6 == 1 {
		msg.IsPostPolicy = true
	} else {
		msg.IsPostPolicy = false
	}
	msg.PeerDistinguisher = binary.BigEndian.Uint64(data[2:10])
	if flags&1<<7 == 1 {
		msg.PeerAddress = data[10:26]
	} else {
		msg.PeerAddress = data[10:14]
	}
	msg.PeerAS = binary.BigEndian.Uint32(data[26:30])
	msg.PeerBGPID = data[30:34]

	timestamp1 := binary.BigEndian.Uint32(data[34:38])
	timestamp2 := binary.BigEndian.Uint32(data[38:42])
	msg.Timestamp = float64(timestamp1) + float64(timestamp2)*math.Pow(10, -6)

	return nil
}

type BMPRouteMonitoring struct {
	BGPUpdate *BGPMessage
}

func (body *BMPRouteMonitoring) ParseBody(msg *BMPMessage, data []byte) error {
	update, err := ParseBGPMessage(data)
	if err != nil {
		return err
	}
	body.BGPUpdate = update
	return nil
}

const (
	BMP_STAT_TYPE_REJECTED = iota
	BMP_STAT_TYPE_DUPLICATE_PREFIX
	BMP_STAT_TYPE_DUPLICATE_WITHDRAW
	BMP_STAT_TYPE_INV_UPDATE_DUE_TO_CLUSTER_LIST_LOOP
	BMP_STAT_TYPE_INV_UPDATE_DUE_TO_AS_PATH_LOOP
	BMP_STAT_TYPE_INV_UPDATE_DUE_TO_ORIGINATOR_ID
	BMP_STAT_TYPE_INV_UPDATE_DUE_TO_AS_CONFED_LOOP
	BMP_STAT_TYPE_ADJ_RIB_IN
	BMP_STAT_TYPE_LOC_RIB
)

type BMPStatsTLV struct {
	Type   uint16
	Length uint16
	Value  uint64
}

type BMPStatisticsReport struct {
	Stats []BMPStatsTLV
}

const (
	BMP_PEER_DOWN_REASON_UNKNOWN = iota
	BMP_PEER_DOWN_REASON_LOCAL_BGP_NOTIFICATION
	BMP_PEER_DOWN_REASON_LOCAL_NO_NOTIFICATION
	BMP_PEER_DOWN_REASON_REMOTE_BGP_NOTIFICATION
	BMP_PEER_DOWN_REASON_REMOTE_NO_NOTIFICATION
)

type BMPPeerDownNotification struct {
	Reason          uint8
	BGPNotification *BGPMessage
	Data            []byte
}

func (body *BMPPeerDownNotification) ParseBody(msg *BMPMessage, data []byte) error {
	body.Reason = data[0]
	data = data[1:]
	if body.Reason == BMP_PEER_DOWN_REASON_LOCAL_BGP_NOTIFICATION || body.Reason == BMP_PEER_DOWN_REASON_REMOTE_BGP_NOTIFICATION {
		notification, err := ParseBGPMessage(data)
		if err != nil {
			return err
		}
		body.BGPNotification = notification
	} else {
		body.Data = data
	}
	return nil
}

type BMPPeerUpNotification struct {
	LocalAddress    net.IP
	LocalPort       uint16
	RemotePort      uint16
	SentOpenMsg     *BGPMessage
	ReceivedOpenMsg *BGPMessage
}

func (body *BMPPeerUpNotification) ParseBody(msg *BMPMessage, data []byte) error {
	if msg.PeerHeader.flags&1<<7 == 1 {
		body.LocalAddress = data[:16]
	} else {
		body.LocalAddress = data[:4]
	}

	body.LocalPort = binary.BigEndian.Uint16(data[16:18])
	body.RemotePort = binary.BigEndian.Uint16(data[18:20])

	data = data[20:]
	sentopen, err := ParseBGPMessage(data)
	if err != nil {
		return err
	}
	body.SentOpenMsg = sentopen
	data = data[body.SentOpenMsg.Header.Len:]
	body.ReceivedOpenMsg, err = ParseBGPMessage(data)
	if err != nil {
		return err
	}
	return nil
}

func (body *BMPStatisticsReport) ParseBody(msg *BMPMessage, data []byte) error {
	_ = binary.BigEndian.Uint32(data[0:4])
	data = data[4:]
	for len(data) >= 4 {
		s := BMPStatsTLV{}
		s.Type = binary.BigEndian.Uint16(data[0:2])
		s.Length = binary.BigEndian.Uint16(data[2:4])

		if s.Type == BMP_STAT_TYPE_ADJ_RIB_IN || s.Type == BMP_STAT_TYPE_LOC_RIB {
			s.Value = binary.BigEndian.Uint64(data[4:12])
		} else {
			s.Value = uint64(binary.BigEndian.Uint32(data[4:8]))
		}
		body.Stats = append(body.Stats, s)
		data = data[4+s.Length:]
	}
	return nil
}

type BMPTLV struct {
	Type   uint16
	Length uint16
	Value  []byte
}

type BMPInitiation struct {
	Info []BMPTLV
}

func (body *BMPInitiation) ParseBody(msg *BMPMessage, data []byte) error {
	for len(data) >= 4 {
		tlv := BMPTLV{}
		tlv.Type = binary.BigEndian.Uint16(data[0:2])
		tlv.Length = binary.BigEndian.Uint16(data[2:4])
		tlv.Value = data[4 : 4+tlv.Length]

		body.Info = append(body.Info, tlv)
		data = data[4+tlv.Length:]
	}
	return nil
}

type BMPTermination struct {
	Info []BMPTLV
}

func (body *BMPTermination) ParseBody(msg *BMPMessage, data []byte) error {
	for len(data) >= 4 {
		tlv := BMPTLV{}
		tlv.Type = binary.BigEndian.Uint16(data[0:2])
		tlv.Length = binary.BigEndian.Uint16(data[2:4])
		tlv.Value = data[4 : 4+tlv.Length]

		body.Info = append(body.Info, tlv)
		data = data[4+tlv.Length:]
	}
	return nil
}

type BMPBody interface {
	ParseBody(*BMPMessage, []byte) error
}

type BMPMessage struct {
	Header     BMPHeader
	PeerHeader BMPPeerHeader
	Body       BMPBody
}

func (msg *BMPMessage) Len() int {
	return int(msg.Header.Length)
}

const (
	BMP_MSG_ROUTE_MONITORING = iota
	BMP_MSG_STATISTICS_REPORT
	BMP_MSG_PEER_DOWN_NOTIFICATION
	BMP_MSG_PEER_UP_NOTIFICATION
	BMP_MSG_INITIATION
	BMP_MSG_TERMINATION
)

// move somewhere else
func ReadBMPMessage(conn net.Conn) (*BMPMessage, error) {
	buf := make([]byte, BMP_HEADER_SIZE)
	for offset := 0; offset < BMP_HEADER_SIZE; {
		rlen, err := conn.Read(buf[offset:])
		if err != nil {
			return nil, err
		}
		offset += rlen
	}

	h := BMPHeader{}
	err := h.DecodeFromBytes(buf)
	if err != nil {
		return nil, err
	}

	data := make([]byte, h.Len())
	copy(data, buf)
	data = data[BMP_HEADER_SIZE:]
	for offset := 0; offset < h.Len()-BMP_HEADER_SIZE; {
		rlen, err := conn.Read(data[offset:])
		if err != nil {
			return nil, err
		}
		offset += rlen
	}
	msg := &BMPMessage{Header: h}

	switch msg.Header.Type {
	case BMP_MSG_ROUTE_MONITORING:
		msg.Body = &BMPRouteMonitoring{}
	case BMP_MSG_STATISTICS_REPORT:
		msg.Body = &BMPStatisticsReport{}
	case BMP_MSG_PEER_DOWN_NOTIFICATION:
		msg.Body = &BMPPeerDownNotification{}
	case BMP_MSG_PEER_UP_NOTIFICATION:
		msg.Body = &BMPPeerUpNotification{}
	case BMP_MSG_INITIATION:
		msg.Body = &BMPInitiation{}
	case BMP_MSG_TERMINATION:
		msg.Body = &BMPTermination{}
	}

	if msg.Header.Type != BMP_MSG_INITIATION && msg.Header.Type != BMP_MSG_INITIATION {
		msg.PeerHeader.DecodeFromBytes(data)
		data = data[42:]
	}

	err = msg.Body.ParseBody(msg, data)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func ParseBMPMessage(data []byte) (*BMPMessage, error) {
	msg := &BMPMessage{}
	msg.Header.DecodeFromBytes(data)
	data = data[6:msg.Header.Length]

	switch msg.Header.Type {
	case BMP_MSG_ROUTE_MONITORING:
		msg.Body = &BMPRouteMonitoring{}
	case BMP_MSG_STATISTICS_REPORT:
		msg.Body = &BMPStatisticsReport{}
	case BMP_MSG_PEER_DOWN_NOTIFICATION:
		msg.Body = &BMPPeerDownNotification{}
	case BMP_MSG_PEER_UP_NOTIFICATION:
		msg.Body = &BMPPeerUpNotification{}
	case BMP_MSG_INITIATION:
		msg.Body = &BMPInitiation{}
	case BMP_MSG_TERMINATION:
		msg.Body = &BMPTermination{}
	}

	if msg.Header.Type != BMP_MSG_INITIATION && msg.Header.Type != BMP_MSG_INITIATION {
		msg.PeerHeader.DecodeFromBytes(data)
		data = data[42:]
	}

	err := msg.Body.ParseBody(msg, data)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
