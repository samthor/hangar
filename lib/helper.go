package lib

import (
	"encoding/json"
	"strings"
)

// parseListRecord parses a record like "key=value,foo=bar;key=value,foo=bar" into an array of those objects.
func parseListRecord(raw string, into interface{}) error {
	configs := strings.Split(raw, ";")
	out := []map[string]string{}

	for _, raw := range configs {
		m := make(map[string]string)
		out = append(out, m)

		parts := strings.Split(raw, ",")
		for _, p := range parts {
			raw := strings.SplitN(p, "=", 2)
			if len(raw) == 2 {
				m[raw[0]] = raw[1]
			}
		}
	}

	// TODO: be less lazy
	tmp, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return json.Unmarshal(tmp, into)
}
