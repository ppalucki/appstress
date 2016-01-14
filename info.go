package main

import (
	"encoding/json"
	"log"
	"time"
)

func storeInfo(interval time.Duration) {
	for {
		i := info()
		if i != nil {
			// TODO: fds/threads
			d := map[string]interface{}{"containers": i["Containers"], "ngoroutines": i["NGoroutines"]}
			log.Println("info = ", d)
			store("info", nil, d)
		}
		time.Sleep(interval)
	}
}

func dumpInfo(i map[string]string) {
	mb, err := json.MarshalIndent(i, "", "  ")
	ok(err)
	log.Println(string(mb))
}
