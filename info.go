package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/ppalucki/dockerstress/influx"
)

func storeInfo() {
	for {
		i := info()
		if i != nil {
			influx.Store("info", nil, map[string]interface{}{"containers": i["Containers"], "ngoroutines": i["NGoroutines"]}, time.Now())
		}
		time.Sleep(STORE)
	}
}

func printInfo() {
	i := info()
	mb, err := json.MarshalIndent(i, "", "  ")
	ok(err)
	log.Println(string(mb))
}
