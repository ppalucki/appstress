package main

import (
	"strings"
	"time"
)

func toFields(s map[string]int) map[string]interface{} {

	data := map[string]int{}
	for k, v := range s {
		k = strings.Split(k, " ")[0]
		prev, ok := data[k]
		if !ok {
			prev = 0
		}
		data[k] = prev + v
	}

	fields := map[string]interface{}{}
	for k, v := range data {
		fields[k] = v
	}
	if len(fields) == 0 {
		fields["up"] = 0
		fields["exited"] = 0
	}
	return fields
}

func writeStatuses() {
	fields := toFields(statuses(true))
	store("statuses", nil, fields)
}

func storeStatuses(interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		for range t.C {
			writeStatuses()
		}
	}()
}
