package main

import (
	"log"
	"strings"
	"time"

	"github.com/ppalucki/dockerstress/influx"
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

func printStatuses() {
	fields := toFields(statuses(true))
	log.Printf("fields = %#v\n", fields)
}

func reportStatuses() {
	go func() {
		printStatuses()
		time.Sleep(REPORT * time.Second)
	}()
}

func storeStatuses() {
	go func() {
		fields := toFields(statuses(true))
		influx.Store("statuses", nil, fields, time.Now())
		time.Sleep(REPORT * time.Second)
	}()
}
