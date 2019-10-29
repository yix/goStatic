package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	StatusText string
)

func StatusInit() {
	var envHints []string
	for _, val := range strings.Split(*statusEnvConfig, ",") {
		envHints = append(envHints, strings.Trim(val, " \t"))
	}
	StatusSetText(envHints)
}

func StatusSetText(envHints []string) {
	sm := StatusGetEnv(envHints)
	if *statusTimestamp {
		sm["startTimestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05.999999Z07:00")
	}
	StatusText = StatusGetJson(sm)
}

func StatusGetEnv(vars []string) map[string]string {
	var resp = make(map[string]string)
	for _, v := range vars {
		s := strings.Split(v, ":")
		switch len(s) {
		case 1:
			resp[v] = os.Getenv(v)
		case 2:
			resp[s[1]] = os.Getenv(s[0])
		default:
			continue
		}
	}
	return resp
}

func StatusGetJson(s map[string]string) string {
	buf, _ := json.Marshal(s)
	return string(buf)
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, StatusText)
}
