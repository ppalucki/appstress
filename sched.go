package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hpcloud/tail"
)

func storeLine(line *tail.Line) {
	tags := make(map[string]string)
	// when = line.Time
	text := line.Text
	if strings.Contains(text, "SCHED") {
		data := strings.Split(text, " ")
		if data[0] != "SCHED" {
			log.Fatal("unrecognized sched trace data: ", text)
		}

		//SCHED 0ms: 2.gomaxprocs=1 idleprocs=0 threads=2 spinningthreads=0 idlethreads=0 runqueue=0 [1]
		gomaxprocs, err := strconv.Atoi(strings.Split(data[2], "=")[1])
		if err != nil {
			log.Fatal(err)
		}
		idleprocs, err := strconv.Atoi(strings.Split(data[3], "=")[1])
		if err != nil {
			log.Fatal(err)
		}
		threads, err := strconv.Atoi(strings.Split(data[4], "=")[1])
		if err != nil {
			log.Fatal(err)
		}
		spinningthreads, err := strconv.Atoi(strings.Split(data[5], "=")[1])
		if err != nil {
			log.Fatal(err)
		}
		idlethreads, err := strconv.Atoi(strings.Split(data[6], "=")[1])
		if err != nil {
			log.Fatal(err)
		}
		runqueue, err := strconv.Atoi(strings.Split(data[7], "=")[1])
		if err != nil {
			log.Fatal(err)
		}

		tags["kind"] = "metrics"
		store("sched", tags,
			map[string]interface{}{
				"gomaxprocs": gomaxprocs, "idleprocs": idleprocs, "threads": threads, "spinningthreads": spinningthreads, "idlethreads": idlethreads, "runqueue": runqueue},
		)
	}

	if strings.Contains("ERRO[", text) {
		fmt.Println("ERROR:", text)
		tags["kind"] = "error"
		store("sched", tags, map[string]interface{}{"message": text, "count": 1, "tags": "error"})
	}

	if strings.Contains("WARN[", text) {
		fmt.Println("WARNING:", text)
		tags["kind"] = "warn"
		store("sched", tags, map[string]interface{}{"message": text, "count": 1, "tags": "warn"})
	}
}

func storeSched(dockerLog string) {

	t, err := tail.TailFile(dockerLog, tail.Config{Follow: true, Location: &tail.SeekInfo{Offset: 0, Whence: 2}})
	ok(err)

	for {
		select {
		case line := <-t.Lines:
			storeLine(line)
		case <-quit:
			return
		}
	}
}
