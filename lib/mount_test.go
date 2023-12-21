package lib

import (
	"fmt"
	"os"
	"testing"
)

func TestResolveStoragePath(t *testing.T) {
	actual := resolveStoragePath("../zing/foo/..")

	expected := fmt.Sprintf("%s/.fly/hangar/%s/mount/zing", os.Getenv("HOME"), selfInstance.Machine)

	if actual != expected {
		t.Errorf("actual=%v expected=%v", actual, expected)
	}
}
