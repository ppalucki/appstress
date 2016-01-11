package main

import (
	"encoding/json"
	"log"
	"time"
)

func storeInfo() {
	for {
		i := info()
		if i != nil {
			store("info", nil, map[string]interface{}{"containers": i["Containers"], "ngoroutines": i["NGoroutines"]})
		}
		time.Sleep(REPORT)
	}
}

func printInfo() {
	i := info()
	mb, err := json.MarshalIndent(i, "", "  ")
	ok(err)
	log.Println(string(mb))
}
