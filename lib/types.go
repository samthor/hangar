package lib

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const (
	PortRange = 128
)

type InstanceInfo struct {
	Machine      string `json:"machine"`
	Region       string `json:"region"`
	Address      string `json:"address"`
	Port         uint16 `json:"port"`
	fallbackSelf bool
}

func (i *InstanceInfo) AddrOffset(offset uint16) netip.AddrPort {
	if offset >= PortRange {
		panic(fmt.Sprintf("cannot offset <0 or >=%d", PortRange))
	}

	addr := netip.MustParseAddr(i.Address)
	return netip.AddrPortFrom(addr, i.Port+offset)
}

func (i *InstanceInfo) Addr() netip.AddrPort {
	return i.AddrOffset(0)
}

func (i *InstanceInfo) IsSelf() bool {
	if i.fallbackSelf {
		return true
	} else if localMachine != "" {
		return i.Machine == localMachine
	} else if flyMachine != "" {
		return i.Machine == flyMachine
	}
	return false
}

type ControlInfo struct {
	Now       int64                   `json:"now"`
	Instances map[string]InstanceInfo `json:"instances"`
	Unknown   bool                    `json:"unknown,omitempty"` // whether there was a (hopefully) transient error in fetching
}

// Peers returns the map of peer address/port combos.
func (ci *ControlInfo) PeerAddrOffset(offset uint16) map[string]netip.AddrPort {
	out := make(map[string]netip.AddrPort)
	for _, i := range ci.Instances {
		out[i.Machine] = i.AddrOffset(offset)
	}
	return out
}

// ByAddr returns the InstanaceInfo by the given address (tcp or udp).
func (ci *ControlInfo) ByAddr(addr net.Addr) *InstanceInfo {
	var ap netip.AddrPort

	if tcp, ok := addr.(*net.TCPAddr); ok {
		ap = tcp.AddrPort()
	} else if udp, ok := addr.(*net.UDPAddr); ok {
		ap = udp.AddrPort()
	}

	return ci.ByAddrPort(ap)
}

// ByAddrPort returns the InstanceInfo by the given address and port.
func (ci *ControlInfo) ByAddrPort(ap netip.AddrPort) *InstanceInfo {
	ip := ap.Addr().String()
	port := ap.Port()

	if ip == "" {
		return nil
	}

	for machineId := range ci.Instances {
		// TODO: lots of memcpy
		i := ci.Instances[machineId]
		if i.Address != ip {
			continue
		}

		if flyMachine != "" {
			return &i // just match machine in real world
		}

		minPort := i.Port
		maxPort := i.Port + PortRange
		if port >= minPort && port < maxPort {
			return &i
		}
	}

	return nil
}

type FlyReplayHeader struct {
	Region    string `json:"region,omitempty"`
	Instance  string `json:"instance,omitempty"`
	App       string `json:"app,omitempty"`
	State     string `json:"state,omitempty"`
	Elsewhere bool   `json:"elsewhere,omitempty"`
	Now       int64  `json:"t,omitempty"`
}

// ForResponseHeader writes this FlyReplayHeader for your userspace code.
func (fr *FlyReplayHeader) ForResponseHeader() string {
	parts := map[string]string{
		"instance":  fr.Instance,
		"region":    fr.Region,
		"app":       fr.App,
		"state":     fr.State,
		"elsewhere": "",
	}
	if fr.Elsewhere {
		parts["elsewhere"] = "true"
	}

	var out []string
	for key, value := range parts {
		if value != "" {
			out = append(out, fmt.Sprintf("%s=%s", key, value))
		}
	}
	return strings.Join(out, ";")
}

// SetResponse applies this FlyReplayHeader to a response for your userspace code.
func (fr *FlyReplayHeader) SetResponse(h http.Header) bool {
	v := fr.ForResponseHeader()
	if v != "" {
		h.Set("fly-replay", v)
		return true
	}
	return false
}
