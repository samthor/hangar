package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"

	mesh "github.com/samthor/hangar/lib"
)

type Instance struct {
	ControlPort uint16
	Port        uint16
	Package     string
	Region      string
	MachineId   string

	active atomic.Int32 // active requests
	lock   sync.RWMutex
	runCh  <-chan *exec.ExitError
}

func (i *Instance) Requests() int {
	return int(i.active.Load())
}

func (i *Instance) Active() bool {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.runCh != nil
}

func (i *Instance) run() <-chan *exec.ExitError {
	closeCh := make(chan *exec.ExitError)

	controlUrl := fmt.Sprintf("http://localhost:%d/__/control?machine=%s", i.ControlPort, i.MachineId)

	e := exec.Command("go", "run", i.Package)
	e.Env = append(
		os.Environ(),
		fmt.Sprintf("PORT=%d", i.Port),
		fmt.Sprintf("MAXPORT=%d", i.Port+mesh.PortRange),
		fmt.Sprintf("LOCAL_CONTROL_URL=%s", controlUrl),
		fmt.Sprintf("LOCAL_MACHINE_ID=%s", i.MachineId),
		fmt.Sprintf("LOCAL_REGION=%s", i.Region),
	)

	// TODO: prefix output
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr

	err := e.Start()
	if err != nil {
		log.Printf("could not run instance=%+v, err=%v", i, err)
	}
	log.Printf("machine=%s running (region=%s, port=%d)", i.MachineId, i.Region, i.Port)

	go func() {
		err := e.Wait()
		if exitErr, ok := err.(*exec.ExitError); ok || err == nil {
			closeCh <- exitErr
		} else {
			log.Fatalf("unhandled err from exec: %v", err)
		}
		close(closeCh)
	}()

	return closeCh
}

func (i *Instance) MatchRegion(region string) bool {
	return region == "" || i.Region == region
}

func (i *Instance) EnsureRun() bool {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.runCh != nil {
		return false
	}

	ch := i.run()
	i.runCh = ch
	go func() {
		err := <-ch
		var exitCode int
		if err != nil {
			exitCode = err.ExitCode()
		}
		log.Printf("machine=%s stopped: %d", i.MachineId, exitCode)

		i.lock.Lock()
		defer i.lock.Unlock()
		i.runCh = nil
	}()

	return true
}

func (i *Instance) IsAlive() bool {
	i.lock.RLock()
	defer i.lock.RUnlock()
	return i.runCh != nil
}

type ErrReplay struct {
	Replay string
}

func (e *ErrReplay) Error() string {
	return fmt.Sprintf("replay: %v", e.Replay)
}

func (i *Instance) SendTo(replay func(i *Instance, replay string), w http.ResponseWriter, r *http.Request) bool {
	var isRefused bool
	i.active.Add(+1)
	defer i.active.Add(-1)

	rp := httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.Host = fmt.Sprintf("localhost:%d", i.Port)
			r.URL.Host = r.Host
			r.URL.Scheme = "http"
		},

		ModifyResponse: func(r *http.Response) error {
			replay := r.Header.Get(headerReplay)
			if replay != "" {
				// TODO: we support region replay, actual Fly supports a lot more
				return &ErrReplay{Replay: replay}
			}

			return nil
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if re, ok := err.(*ErrReplay); ok {
				// http.Error(w, "replay:"+replay.Replay, http.StatusTeapot)
				replay(i, re.Replay)
				return
			}

			isRefused = errors.Is(err, syscall.ECONNREFUSED)
			if !isRefused {
				log.Printf("got gatway err: %+v", err)
				http.Error(w, "", http.StatusBadGateway)
			}
		},
	}
	rp.ServeHTTP(w, r)

	return !isRefused
}

func (i *Instance) Less(other *Instance) bool {
	active := i.Active()
	otherActive := other.Active()

	if active != otherActive {
		return active // active insts are higher priority
	}

	requests := i.Requests()
	otherRequests := other.Requests()

	if requests != otherRequests {
		return requests < otherRequests // we have fewer requests
	}

	return false
}

type InstanceList []*Instance

func (il InstanceList) Len() int           { return len(il) }
func (il InstanceList) Less(i, j int) bool { return il[i].Less(il[j]) }
func (il InstanceList) Swap(i, j int)      { il[i], il[j] = il[j], il[i] }
