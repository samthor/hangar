package lib

import (
	"fmt"
	"os"
	"strconv"
)

var (
	localMachine    = os.Getenv("LOCAL_MACHINE_ID")
	localControlUrl = os.Getenv("LOCAL_CONTROL_URL")
	localRegion     = os.Getenv("LOCAL_REGION")
	flyMachine      = os.Getenv("FLY_MACHINE_ID")
	flyProcessGroup = os.Getenv("FLY_PROCESS_GROUP")
	flyAppName      = os.Getenv("FLY_APP_NAME")
	flyRegion       = os.Getenv("FLY_REGION")
)

var (
	selfInstance InstanceInfo
)

func init() {
	portRaw, _ := strconv.Atoi(os.Getenv("PORT"))
	port := uint16(portRaw)
	if port == 0 {
		port = flyDefaultPort
	}

	if flyMachine != "" {
		if port != flyDefaultPort {
			panic(fmt.Sprintf("cannot mesh without PORT=%d", flyDefaultPort))
		}

		selfInstance = InstanceInfo{
			Machine: flyMachine,
			Region:  flyRegion,
			Address: os.Getenv("FLY_PRIVATE_IP"),
			Port:    flyDefaultPort,
		}
	} else if localMachine != "" {
		selfInstance = InstanceInfo{
			Machine: localMachine,
			Region:  localRegion,
			Address: "::1",
			Port:    port,
		}
	} else {
		selfInstance = InstanceInfo{
			Machine: "zzxxzzxx",
			Region:  "qqq",
			Address: "::1",
			Port:    port,
		}
	}

	selfInstance.fallbackSelf = true
}
