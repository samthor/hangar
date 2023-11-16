package lib

import (
	"testing"
)

func TestInstanceInfo(t *testing.T) {
	i := InstanceInfo{
		Machine: "cafe0000",
	}

	u, ok := i.AsUint()
	if !ok || u != 0xcafe0000 {
		t.Fatalf("cooerce expected=%v was=%v", 0xcafe0000, u)
	}
}
