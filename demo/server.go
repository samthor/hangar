package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	mesh "github.com/samthor/hangar/lib"
)

const (
	secretOffset = 1
)

func main() {
	self := mesh.Self()
	secretCode := rand.Int63()

	log.Printf("startup inst=%+v our secret=%d", self, secretCode)

	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		ci, err := mesh.Discover(r.Context())
		if err != nil {
			log.Printf("could not discover: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		fmt.Fprintf(w, "Hello user, self=%+v\n", self)

		fmt.Fprintf(w, "-- %d active remotes\n", len(ci.Instances))
		for _, instance := range ci.Instances {
			remoteSecretCode, err := secretFromRemote(&instance)
			if err != nil {
				fmt.Fprintf(w, ".. inst=%s err=%v\n", instance.Machine, err)
			} else {
				fmt.Fprintf(w, ".. inst=%s secret=%v\n", instance.Machine, remoteSecretCode)
			}
		}
	})

	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Ok, shutting down gracefully")
		go func() {
			time.Sleep(time.Millisecond * 10)
			os.Exit(0)
		}()
	})

	http.HandleFunc("/die", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Ok, shutting down with error code")
		go func() {
			time.Sleep(time.Millisecond * 10)
			os.Exit(1)
		}()
	})

	go func() {
		// run public server
		log.Fatal(http.ListenAndServe(mesh.ListenPort(), nil))
	}()

	internalMux := http.NewServeMux()
	internalMux.HandleFunc("/secret", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%d", secretCode)
	})

	// run internal server (not on PORT, so not public)
	log.Fatal(http.ListenAndServe(mesh.ListenPortOffset(secretOffset), internalMux))
}

func secretFromRemote(i *mesh.InstanceInfo) (int64, error) {
	remoteUrl := fmt.Sprintf("http://[%s]:%d/secret", i.Address, i.Port+secretOffset)
	log.Printf("dialing: %s", remoteUrl)
	r, err := http.Get(remoteUrl)
	if err != nil {
		return 0, err
	}
	defer r.Body.Close()
	value, err := io.ReadAll(r.Body)
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(string(value), 10, 64)
}
