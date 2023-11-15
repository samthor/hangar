package lib

import (
	"reflect"
	"testing"
)

func TestParseRecord(t *testing.T) {
	raw := "ip=123,foo=bar;ip=456"

	type Data struct {
		Ip  string `json:"ip"`
		Foo string `json:"foo"`
	}

	var out []Data
	err := parseListRecord(raw, &out)
	if err != nil {
		t.Errorf("could not parse: %v", err)
	}

	if expected := []Data{
		{Ip: "123", Foo: "bar"},
		{Ip: "456"},
	}; !reflect.DeepEqual(out, expected) {
		t.Errorf("invalid match: wanted=%v, got=%v", out, expected)
	}
}
