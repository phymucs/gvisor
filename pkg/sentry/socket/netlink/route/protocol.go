// Copyright 2018 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package route provides a NETLINK_ROUTE socket protocol.
package route

import (
	"bytes"
	"syscall"

	"gvisor.dev/gvisor/pkg/abi/linux"
	"gvisor.dev/gvisor/pkg/sentry/context"
	"gvisor.dev/gvisor/pkg/sentry/inet"
	"gvisor.dev/gvisor/pkg/sentry/kernel"
	"gvisor.dev/gvisor/pkg/sentry/kernel/auth"
	"gvisor.dev/gvisor/pkg/sentry/socket/netlink"
	"gvisor.dev/gvisor/pkg/syserr"
)

// commandKind describes the operational class of a message type.
//
// The route message types use the lower 2 bits of the type to describe class
// of command.
type commandKind int

const (
	kindNew commandKind = 0x0
	kindDel             = 0x1
	kindGet             = 0x2
	kindSet             = 0x3
)

func typeKind(typ uint16) commandKind {
	return commandKind(typ & 0x3)
}

// Protocol implements netlink.Protocol.
//
// +stateify savable
type Protocol struct{}

var _ netlink.Protocol = (*Protocol)(nil)

// NewProtocol creates a NETLINK_ROUTE netlink.Protocol.
func NewProtocol(t *kernel.Task) (netlink.Protocol, *syserr.Error) {
	return &Protocol{}, nil
}

// Protocol implements netlink.Protocol.Protocol.
func (p *Protocol) Protocol() int {
	return linux.NETLINK_ROUTE
}

// CanSend implements netlink.Protocol.CanSend.
func (p *Protocol) CanSend() bool {
	return true
}

// dumpLinks handles RTM_GETLINK + NLM_F_DUMP requests.
func (p *Protocol) dumpLinks(ctx context.Context, msg *netlink.Message, ms *netlink.MessageSet) *syserr.Error {
	// NLM_F_DUMP + RTM_GETLINK messages are supposed to include an
	// ifinfomsg. However, Linux <3.9 only checked for rtgenmsg, and some
	// userspace applications (including glibc) still include rtgenmsg.
	// Linux has a workaround based on the total message length.
	//
	// We don't bother to check for either, since we don't support any
	// extra attributes that may be included anyways.
	//
	// The message may also contain netlink attribute IFLA_EXT_MASK, which
	// we don't support.

	// The RTM_GETLINK dump response is a set of messages each containing
	// an InterfaceInfoMessage followed by a set of netlink attributes.

	// We always send back an NLMSG_DONE.
	ms.Multi = true

	stack := inet.StackFromContext(ctx)
	if stack == nil {
		// No network devices.
		return nil
	}

	for idx, i := range stack.Interfaces() {
		addNewLinkMessage(ms, idx, i)
	}

	return nil
}

// getLinks handles RTM_GETLINK requests.
func (p *Protocol) getLink(ctx context.Context, msg *netlink.Message, ms *netlink.MessageSet) *syserr.Error {
	stack := inet.StackFromContext(ctx)
	if stack == nil {
		// No network devices.
		return nil
	}

	// Parse message.
	var ifi linux.InterfaceInfoMessage
	attrs, ok := msg.GetData(&ifi)
	if !ok {
		return syserr.ErrInvalidArgument
	}

	// Parse attributes.
	var byName []byte
	for !attrs.Empty() {
		ahdr, value, rest, ok := attrs.Next()
		if !ok {
			return syserr.ErrInvalidArgument
		}
		attrs = rest

		switch ahdr.Type {
		case linux.IFLA_IFNAME:
			if len(value) < 1 {
				return syserr.ErrInvalidArgument
			}
			byName = value[:len(value)-1]

			// TODO: Support IFLA_EXT_MASK.
		}
	}

	found := false
	for idx, i := range stack.Interfaces() {
		switch {
		case ifi.Index > 0:
			if idx != ifi.Index {
				continue
			}
		case byName != nil:
			if string(byName) != i.Name {
				continue
			}
		default:
			// Criteria not specified.
			return syserr.ErrInvalidArgument
		}

		addNewLinkMessage(ms, idx, i)
		found = true
		break
	}
	if !found {
		return syserr.ErrNoDevice
	}
	return nil
}

// addNewLinkMessage appends RTM_NEWLINK message for the given interface into
// the message set.
func addNewLinkMessage(ms *netlink.MessageSet, idx int32, i inet.Interface) {
	m := ms.AddMessage(linux.NetlinkMessageHeader{
		Type: linux.RTM_NEWLINK,
	})

	m.Put(linux.InterfaceInfoMessage{
		Family: linux.AF_UNSPEC,
		Type:   i.DeviceType,
		Index:  idx,
		Flags:  i.Flags,
	})

	m.PutAttrString(linux.IFLA_IFNAME, i.Name)
	m.PutAttr(linux.IFLA_MTU, i.MTU)

	mac := make([]byte, 6)
	brd := mac
	if len(i.Addr) > 0 {
		mac = i.Addr
		brd = bytes.Repeat([]byte{0xff}, len(i.Addr))
	}
	m.PutAttr(linux.IFLA_ADDRESS, mac)
	m.PutAttr(linux.IFLA_BROADCAST, brd)

	// TODO(gvisor.dev/issue/578): There are many more attributes.
}

// dumpAddrs handles RTM_GETADDR + NLM_F_DUMP requests.
func (p *Protocol) dumpAddrs(ctx context.Context, msg *netlink.Message, ms *netlink.MessageSet) *syserr.Error {
	// RTM_GETADDR dump requests need not contain anything more than the
	// netlink header and 1 byte protocol family common to all
	// NETLINK_ROUTE requests.
	//
	// TODO(b/68878065): Filter output by passed protocol family.

	// The RTM_GETADDR dump response is a set of RTM_NEWADDR messages each
	// containing an InterfaceAddrMessage followed by a set of netlink
	// attributes.

	// We always send back an NLMSG_DONE.
	ms.Multi = true

	stack := inet.StackFromContext(ctx)
	if stack == nil {
		// No network devices.
		return nil
	}

	for id, as := range stack.InterfaceAddrs() {
		for _, a := range as {
			m := ms.AddMessage(linux.NetlinkMessageHeader{
				Type: linux.RTM_NEWADDR,
			})

			m.Put(linux.InterfaceAddrMessage{
				Family:    a.Family,
				PrefixLen: a.PrefixLen,
				Index:     uint32(id),
			})

			m.PutAttr(linux.IFA_ADDRESS, []byte(a.Addr))

			// TODO(gvisor.dev/issue/578): There are many more attributes.
		}
	}

	return nil
}

// dumpRoutes handles RTM_GETROUTE + NLM_F_DUMP requests.
func (p *Protocol) dumpRoutes(ctx context.Context, msg *netlink.Message, ms *netlink.MessageSet) *syserr.Error {
	// RTM_GETROUTE dump requests need not contain anything more than the
	// netlink header and 1 byte protocol family common to all
	// NETLINK_ROUTE requests.

	// We always send back an NLMSG_DONE.
	ms.Multi = true

	stack := inet.StackFromContext(ctx)
	if stack == nil {
		// No network routes.
		return nil
	}

	for _, rt := range stack.RouteTable() {
		m := ms.AddMessage(linux.NetlinkMessageHeader{
			Type: linux.RTM_NEWROUTE,
		})

		m.Put(linux.RouteMessage{
			Family: rt.Family,
			DstLen: rt.DstLen,
			SrcLen: rt.SrcLen,
			TOS:    rt.TOS,

			// Always return the main table since we don't have multiple
			// routing tables.
			Table:    linux.RT_TABLE_MAIN,
			Protocol: rt.Protocol,
			Scope:    rt.Scope,
			Type:     rt.Type,

			Flags: rt.Flags,
		})

		m.PutAttr(254, []byte{123})
		if rt.DstLen > 0 {
			m.PutAttr(linux.RTA_DST, rt.DstAddr)
		}
		if rt.SrcLen > 0 {
			m.PutAttr(linux.RTA_SRC, rt.SrcAddr)
		}
		if rt.OutputInterface != 0 {
			m.PutAttr(linux.RTA_OIF, rt.OutputInterface)
		}
		if len(rt.GatewayAddr) > 0 {
			m.PutAttr(linux.RTA_GATEWAY, rt.GatewayAddr)
		}

		// TODO(gvisor.dev/issue/578): There are many more attributes.
	}

	return nil
}

// newAddr handles RTM_NEWADDR requests.
func (p *Protocol) newAddr(ctx context.Context, msg *netlink.Message, ms *netlink.MessageSet) *syserr.Error {
	stack := inet.StackFromContext(ctx)
	if stack == nil {
		// No network stack.
		return syserr.ErrProtocolNotSupported
	}

	var ifa linux.InterfaceAddrMessage
	attrs, ok := msg.GetData(&ifa)
	if !ok {
		return syserr.ErrInvalidArgument
	}

	for !attrs.Empty() {
		ahdr, value, rest, ok := attrs.Next()
		if !ok {
			return syserr.ErrInvalidArgument
		}
		attrs = rest

		switch ahdr.Type {
		case linux.IFA_ADDRESS:
			err := stack.AddInterfaceAddr(int32(ifa.Index), inet.InterfaceAddr{
				Family:    ifa.Family,
				PrefixLen: ifa.PrefixLen,
				Flags:     ifa.Flags,
				Addr:      value,
			})
			if err == syscall.EEXIST {
				flags := msg.Header().Flags
				if flags&linux.NLM_F_EXCL != 0 {
					return syserr.ErrExists
				}
			} else if err != nil {
				return syserr.ErrInvalidArgument
			}
		}
	}
	return nil
}

// ProcessMessage implements netlink.Protocol.ProcessMessage.
func (p *Protocol) ProcessMessage(ctx context.Context, msg *netlink.Message, ms *netlink.MessageSet) *syserr.Error {
	hdr := msg.Header()

	// All messages start with a 1 byte protocol family.
	var family uint8
	if _, ok := msg.GetData(&family); !ok {
		// Linux ignores messages missing the protocol family. See
		// net/core/rtnetlink.c:rtnetlink_rcv_msg.
		return nil
	}

	// Non-GET message types require CAP_NET_ADMIN.
	if typeKind(hdr.Type) != kindGet {
		creds := auth.CredentialsFromContext(ctx)
		if !creds.HasCapability(linux.CAP_NET_ADMIN) {
			return syserr.ErrPermissionDenied
		}
	}

	if hdr.Flags&linux.NLM_F_DUMP == linux.NLM_F_DUMP {
		// TODO(b/68878065): Only the dump variant of the types below are
		// supported.
		switch hdr.Type {
		case linux.RTM_GETLINK:
			return p.dumpLinks(ctx, msg, ms)
		case linux.RTM_GETADDR:
			return p.dumpAddrs(ctx, msg, ms)
		case linux.RTM_GETROUTE:
			return p.dumpRoutes(ctx, msg, ms)
		default:
			return syserr.ErrNotSupported
		}
	} else if hdr.Flags&linux.NLM_F_REQUEST == linux.NLM_F_REQUEST {
		switch hdr.Type {
		case linux.RTM_GETLINK:
			return p.getLink(ctx, msg, ms)
		case linux.RTM_NEWADDR:
			return p.newAddr(ctx, msg, ms)
		default:
			return syserr.ErrNotSupported
		}
	}
	return syserr.ErrNotSupported
}

// init registers the NETLINK_ROUTE provider.
func init() {
	netlink.RegisterProvider(linux.NETLINK_ROUTE, NewProtocol)
}
