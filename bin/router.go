package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	mesh "github.com/samthor/hangar/lib"
)

const (
	headerPreferRegion = "fly-prefer-region"
	headerReplay       = "fly-replay"
)

type routerState struct {
	replays      int
	replayHeader *mesh.FlyReplayHeader
	target       mesh.FlyReplayHeader
	ro           *Router
	w            http.ResponseWriter
	r            *http.Request
}

func (rs *routerState) Replay(i *Instance, replay string) {
	if rs.replays > *flagReplayCount {
		log.Printf("Got excessively replayed request: url=%v", rs.r.URL)
		http.Error(rs.w, "", http.StatusInternalServerError)
		return
	}
	rs.replays++

	var info mesh.FlyReplayHeader
	parseRecord(replay, &info)

	if info.Instance != "" || info.App != "" || info.Elsewhere {
		log.Printf("Unhandled replay header: %+v", info)
		http.Error(rs.w, "", http.StatusInternalServerError)
		return
	}
	rs.target = info

	// this is "where we were from", not where we're going
	rs.replayHeader = &mesh.FlyReplayHeader{
		Instance: i.MachineId,
		Region:   i.Region,
		State:    info.State,
	}

	rs.ro.serveForRegion(rs, rs.w, rs.r)
}

type Router struct {
	regionToInstance map[string]InstanceList
	defaultRegion    string
}

func (ro *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rs := &routerState{
		ro: ro,
		w:  w,
		r:  r,
		target: mesh.FlyReplayHeader{
			Region: r.Header.Get(headerPreferRegion),
		},
	}
	ro.serveForRegion(rs, w, r)
}

func (ro *Router) serveForRegion(rs *routerState, w http.ResponseWriter, r *http.Request) {
	region := strings.ToLower(strings.TrimSpace(rs.target.Region))

	if region == "" || ro.regionToInstance[region] == nil {
		region = ro.defaultRegion
		if ro.regionToInstance[region] == nil {
			for cand := range ro.regionToInstance {
				region = cand // choose random region
				break
			}
		}
		if region == "" {
			http.Error(w, "", http.StatusBadGateway)
			return
		}
	}

	if rs.replayHeader != nil {
		r.Header.Set("fly-replay-src", replayForRequestHeader(rs.replayHeader))
	}

	options := ro.regionToInstance[region]
	if len(options) == 0 {
		panic(fmt.Sprintf("got invalid region: %s", region))
	}

	// fast-path: find the first region-matched instance handling few requests
	for _, i := range options {
		if i.IsAlive() && i.Requests() < *flagActive && i.SendTo(rs.Replay, w, r) {
			return
		}
	}

	// otherwise, start an instance
	for _, i := range options {
		if !i.EnsureRun() {
			continue // we decided all alive weren't good
		}

		delayPart := healthyTimeout / healthyRetries
		for j := 0; j < healthyRetries; j++ {
			if i.SendTo(rs.Replay, w, r) {
				return
			}
			time.Sleep(delayPart)
		}
	}

	// otherwise, go random
	choice := options[rand.Intn(len(options))]
	if choice.SendTo(rs.Replay, w, r) {
		return
	}

	// can't find a matching region instance
	http.Error(w, "", http.StatusInternalServerError)
}

// ForRequestHeader writes this FlyReplayHeader for the server making a replay request.
func replayForRequestHeader(fr *mesh.FlyReplayHeader) string {
	now := fr.Now
	if now == 0 {
		now = time.Now().UnixMicro()
	}

	out := fmt.Sprintf("instance=%s;region=%s;t=%d", fr.Instance, fr.Region, now)
	if fr.State != "" {
		out += fmt.Sprintf(";state=%s", fr.State)
	}
	return out
}
