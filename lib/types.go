package lib

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
)

const (
	// PortRange is used in dev mode: each instance is given a port range on localhost.
	// Requests for ports via the helpers outside this range will panic.
	PortRange = 128
)

type InstanceInfo struct {
	Machine      string `json:"machine"` // the machine ID reported by Fly
	Region       string `json:"region"`  // the 3-character region code
	Address      string `json:"address"` // the private IPv6 of this instance
	Port         uint16 `json:"port"`    // the default port of this instance
	fallbackSelf bool
}

// Addr returns the netip.AddrPort for the target instance using its default public port.
func (i *InstanceInfo) Addr() netip.AddrPort {
	return i.AddrOffset(0)
}

// AddrOffset returns the netip.AddrPort reflecting the target machine plus a port offset.
func (i *InstanceInfo) AddrOffset(offset uint16) netip.AddrPort {
	if offset >= PortRange {
		panic(fmt.Sprintf("cannot offset <0 or >=%d", PortRange))
	}

	addr := netip.MustParseAddr(i.Address)
	return netip.AddrPortFrom(addr, i.Port+offset)
}

// IsSelf returns whether this InstanceInfo represents the currently running machine.
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

// AsUint attempts to coerce the hex-encoded machine ID into a uint64.
// It's possible that Fly changes the way machines are represented, but as of 2023-11, they are 14-character hex codes (56 bits).
func (i *InstanceInfo) AsUint() (out uint64, ok bool) {
	out, err := strconv.ParseUint(i.Machine, 16, 64)
	if err != nil {
		return 0, false
	}
	return out, true
}

type ControlInfo struct {
	Now       int64          `json:"now"`
	Instances []InstanceInfo `json:"instances"`
	Unknown   bool           `json:"unknown,omitempty"` // whether there was a (hopefully) transient error in fetching
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

	for index := range ci.Instances {
		// TODO: lots of memcpy
		instance := ci.Instances[index]

		if flyMachine != "" {
			// Just match IP in real world, no weird port shenanigans.
			if instance.Address == ip {
				return &instance
			}
		} else {
			// In dev, the instance is matched by its min/max port range (everything is on localhost).
			minPort := instance.Port
			maxPort := instance.Port + PortRange
			if port >= minPort && port < maxPort {
				return &instance
			}
		}
	}

	return nil
}

// FlyReplayHeader can be used to construct a reply that asks another instance to handle a request, rather than handling it directly.
type FlyReplayHeader struct {
	Region    string `json:"region,omitempty"`
	Instance  string `json:"instance,omitempty"`
	App       string `json:"app,omitempty"`
	State     string `json:"state,omitempty"`
	Elsewhere bool   `json:"elsewhere,omitempty"`
	Now       int64  `json:"t,omitempty"`
}

// ForResponseHeader writes this FlyReplayHeader to attach to a header output.
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
			// TODO: what happens if values include =, ; or other unescaped chars
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
