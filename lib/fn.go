package lib

import (
	"fmt"
	"os"
	"strconv"
)

// PrivateHost returns the internal private host that should be listened on for non-public ports.
// This is probably optional, because likely only PORT is exposed to the real world.
func PrivateHost() string {
	if flyMachine != "" {
		return "fly-local-6pn"
	}
	return "localhost"
}

// IsDeploy returns whether this is deployed to Fly.
func IsDeploy() bool {
	return flyMachine != ""
}

// Self immediately returns information about ourself as a mesh instance.
func Self() InstanceInfo {
	return selfInstance
}

// Port returns the primary port of this service. In production this is probably 8080.
func Port() int {
	return int(selfInstance.Port)
}

// ListenPort returns a string for a HTTP server to listen on.
func ListenPort() string {
	return ListenPortOffset(0)
}

// PortOffset returns this port plus an offset.
func PortOffset(offset uint16) int {
	out := selfInstance.Port + offset
	max, _ := strconv.Atoi(os.Getenv("MAXPORT"))
	if max > 0 && out >= uint16(max) {
		panic(fmt.Sprintf("can't assign port %d, >=max %d", out, max))
	}
	return int(out)
}

// ListenPortOffset returns a string for a HTTP server to listen on.
func ListenPortOffset(offset uint16) string {
	var host string
	if IsDeploy() {
		// be explicit for... reasons
		host = "[::]"
	} else {
		// stop egregious firewalls; if you need local dev network access...?
		host = "localhost"
	}
	return fmt.Sprintf("%s:%d", host, PortOffset(offset))
}
