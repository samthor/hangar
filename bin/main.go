// Provides a local daemon for running things like Fly.io servers locally.
// Assumes that the package under control stops after some time (does not kill it).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	mesh "github.com/samthor/hangar/lib"
)

const (
	healthyTimeout = time.Second * 4
	healthyRetries = 12
)

var (
	flagPort        = flag.Uint("port", 8080, "the forward-facing web address")
	flagCount       = flag.Int("c", 4, "number of instances to run")
	flagPackage     = flag.String("p", "", "go package to run")
	flagRegion      = flag.String("r", "syd,ord,ams", "round-robin around these virtual regions")
	flagSeed        = flag.Int64("s", 1, "seed for random machine IDs")
	flagActive      = flag.Int("a", 2, "if handling >requests, try another machine")
	flagReplayCount = flag.Int("replay", 4, "number of times a request can be replayedf")
)

var (
	allInstances []*Instance
)

func main() {
	flag.Parse()
	if *flagPackage == "" {
		log.Fatalf("need -p <package> to run")
	}

	regions := strings.Split(strings.ToLower(*flagRegion), ",")
	if len(regions) == 0 {
		log.Fatalf("need comma-separated regions: had %v", regions)
	}
	for i, r := range regions {
		r = strings.ToLower(strings.TrimSpace(r))
		regions[i] = r
		if len(r) != 3 {
			log.Fatalf("regions must be 3-character codes, had %v", regions)
		}
	}
	defaultRegion := regions[0]
	log.Printf("choosing default region=%s from regions=%v", defaultRegion, regions)

	portStart := *flagPort + 1
	maxPort := portStart + (uint(*flagCount) * mesh.PortRange)
	if maxPort >= 65536 {
		log.Fatalf("can't run %d instances (%d ports each), max=%d", *flagCount, mesh.PortRange, maxPort)
	}

	r := rand.NewSource(*flagSeed)
	router := &Router{
		regionToInstance: make(map[string]InstanceList),
		defaultRegion:    defaultRegion,
	}

	for i := 0; i < *flagCount; i++ {
		port := portStart + uint(i*mesh.PortRange)
		region := regions[i%len(regions)]

		machineNo := uint64(r.Int63()) >> 31
		machineId := fmt.Sprintf("%0x", machineNo)
		for len(machineId) < 8 {
			machineId = "0" + machineId
		}

		i := &Instance{
			ControlPort: uint16(*flagPort),
			Port:        uint16(port),
			Region:      region,
			Package:     *flagPackage,
			MachineId:   machineId,
		}
		router.regionToInstance[region] = append(router.regionToInstance[region], i)
		allInstances = append(allInstances, i)

		log.Printf("generated machine=%s (region=%s port=%d)", i.MachineId, i.Region, port)
	}

	var handler http.ServeMux
	handler.HandleFunc("/__/", handleSpecial)
	handler.Handle("/", router)
	http.ListenAndServe(fmt.Sprintf("localhost:%d", *flagPort), &handler)
}

func handleSpecial(w http.ResponseWriter, r *http.Request) {
	var out interface{}

	switch r.URL.Path {
	case "/__/control":
		out = handleSpecialControl(r)

	case "/__/start":
		out = handleSpecialStart(r)

	default:
		http.Error(w, "", http.StatusNotFound)
	}

	if err, ok := out.(error); ok {
		log.Printf("special err: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
	} else if out != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	}
}

// handleSpecialControl returns an equivalent of Fly's DNS instance discovery.
func handleSpecialControl(r *http.Request) interface{} {
	machine := r.URL.Query().Get("machine")

	c := mesh.ControlInfo{
		Now:       time.Now().UnixMilli(),
		Instances: make(map[string]mesh.InstanceInfo),
	}
	for _, i := range allInstances {
		if !i.IsAlive() || i.MachineId == machine {
			continue // don't include dead or ourselves
		}
		c.Instances[i.MachineId] = mesh.InstanceInfo{
			Machine: i.MachineId,
			Region:  i.Region,
			Address: "::1",
			Port:    i.Port,
		}
	}

	return &c
}

// handleSpecialStart starts all instances immediately.
func handleSpecialStart(r *http.Request) interface{} {
	var changes int
	for _, i := range allInstances {
		if i.EnsureRun() {
			changes++
		}
	}
	return fmt.Sprintf("ok, started %d/%d", changes, len(allInstances))
}
