package main

import (
	"encoding/json"
	"strings"
)

// parseRecord parses a record like "key=value;foo=bar;key=value" into an object.
func parseRecord(raw string, into interface{}) error {
	parts := strings.Split(raw, ";")
	out := map[string]string{}

	for _, p := range parts {
		raw := strings.SplitN(p, "=", 2)
		if len(raw) == 2 {
			out[raw[0]] = raw[1]
		}
	}

	// TODO: be less lazy
	tmp, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return json.Unmarshal(tmp, into)
}
