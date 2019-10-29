package main

import (
	"fmt"
	"os"
	"testing"
)

func TestStatusEnv(t *testing.T) {
	vars := map[string][]string{
		"VERSION":  {"VERSION", "v.1.7.0"},
		"env":      {"ENV", "production"},
		"build_id": {"BUILD", "1337"},
	}
	for key, val := range vars {
		err := os.Setenv(val[0], val[1])
		if err != nil {
			t.Errorf("Setenv failed for %v: %v", key, err)
		}
	}
	var varList []string
	for key, val := range vars {
		if key == val[0] {
			varList = append(varList, key)
		} else {
			varList = append(varList, fmt.Sprintf("%v:%v", val[0], key))
		}
	}
	ev := StatusGetEnv(varList)
	for key, val := range vars {
		t.Logf("%v->%v >>> %v", key, val, ev[key])
		if v, ok := ev[key]; ok {
			if v != val[1] {
				t.Errorf("Wrong value. Got '%v' instead of '%v'", v, val)
			}
		} else {
			t.Errorf("Key missing in resulting map: %v", key)
		}
	}
	js := StatusGetJson(ev)
	jsExpected := `{"VERSION":"v.1.7.0","build_id":"1337","env":"production"}`
	t.Logf("%v", js)
	if js != jsExpected {
		t.Errorf("Unexpected JSON!\nExpected:\n'%v'\nReceived:\n'%v'", jsExpected, js)
	}
}
