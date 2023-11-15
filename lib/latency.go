package lib

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"
)

const (
	maxVirtualLatency = time.Second
)

// VirtualLatency calculates a fake, consistent latency between two named regions.
// It's the same in either direction.
// This always returns zero in deploy contexts.
func VirtualLatency(from, to string) time.Duration {
	if IsDeploy() {
		return 0
	}

	if from == to {
		return 0 // no latency
	} else if from < to {
		from, to = to, from // flip for consistency
	}

	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("%s:%s", from, to)))

	s := rand.NewSource(int64(h.Sum64()))
	r := rand.New(s)

	out := r.Int63n(int64(maxVirtualLatency))
	return time.Duration(out)
}
