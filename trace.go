package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

func getTrace(appUrl string, duration time.Duration) {
	resp, err := http.Get(appUrl + "/debug/pprof/trace?seconds=" + strconv.Itoa(int(duration.Seconds())))
	if warn(err) && resp != nil {
		log.Println("trace download error:")
	}
	f, err := ioutil.TempFile(absDir, "trace-"+*NAME+"-"+time.Now().Format(time.RFC3339)+"-")
	ok(err)
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	ok(err)
	log.Println("trace written:", n, f.Name())
}
