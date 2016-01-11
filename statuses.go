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
	return fields
}

func storeStatuses() {
	go func() {
		fields := toFields(statuses(true))
		if len(fields) == 0 {
			fields["up"] = 0
			fields["exited"] = 0
		}
		store("statuses", nil, fields)
		time.Sleep(REPORT * time.Second)
	}()
}
