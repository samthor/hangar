package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	// assume Fly.io is always on 8080
	flyDefaultPort = uint16(8080)
)

type flyInstance struct {
	Instance     string `json:"instance"` // the ID of the instance
	App          string `json:"app"`
	PrivateIp    string `json:"ip"`
	ProcessGroup string `json:"processGroup"`
	Region       string `json:"region"`
}

// Discover finds mesh instances for this process. This excludes ourselves.
// This takes ~1ms in deploy (DNS lookup) relying on Fly's replication magic.
func Discover(ctx context.Context) (*ControlInfo, error) {
	ci := &ControlInfo{
		Now:       time.Now().UnixMilli(),
		Instances: make(map[string]InstanceInfo),
	}

	// Fetch information from the local controller.
	if localControlUrl != "" {
		resp, err := http.Get(localControlUrl)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var out ControlInfo
		err = json.NewDecoder(resp.Body).Decode(&out)
		return &out, err
	}

	// Fetch information from Fly.
	if flyMachine != "" {
		eg, _ := errgroup.WithContext(ctx)

		ipMap := make(map[string]bool)
		var instances []flyInstance

		eg.Go(func() error {
			// This limits to the actual process group, which _instances does not, below
			ret, err := net.LookupIP(fmt.Sprintf("%s.process.%s.internal", flyProcessGroup, flyAppName))
			if err != nil {
				return err
			}
			for _, ip := range ret {
				ipMap[ip.String()] = true
			}
			return nil
		})
		eg.Go(func() error {
			raw, err := net.LookupTXT("_instances.internal")
			if err != nil || len(raw) == 0 {
				return err
			}
			return parseListRecord(raw[0], &instances)
		})

		err := eg.Wait()
		if err != nil {
			if err, ok := err.(*net.DNSError); ok && err.IsNotFound {
				// this can happen during startup: bail safely
				goto fail
			}
			return nil, err
		}

		for _, fi := range instances {
			if ok := ipMap[fi.PrivateIp]; !ok {
				continue // not part of right process group
			}

			i := InstanceInfo{
				Machine: fi.Instance,
				Region:  fi.Region,
				Address: fi.PrivateIp,
				Port:    flyDefaultPort,
			}

			// check this logic in case we get weird results
			if !i.IsSelf() {
				ci.Instances[i.Machine] = i
			}
		}

		return ci, nil
	}

fail:
	ci.Unknown = true
	return ci, nil
}
