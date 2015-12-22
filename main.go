package main

import (
	"flag"
	"log"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/ppalucki/dockerstress/influx"
)

var (
	DOCKER          = "http://127.0.0.1:8080"
	INFLUX          = "http://127.0.0.1:8086"
	FILE            = "influx.data"
	N               = 100  // in parallel
	B               = 1000 // how many in one batch
	IGNORE_CONFLICT = true
	NAME            = "docker" // measurment name
	IMAGE           = "alpine"
	CMD             = "sleep 8640000"
	REPORT          = time.Duration(1 * time.Second)
	STORE           = time.Duration(1 * time.Second)
	PIDFILE         = "docker.pid"

	c    *docker.Client
	quit chan struct{}
)

func initDocker() {
	// connect docker
	// c, err := docker.NewClientFromEnv()
	var err error
	c, err = docker.NewClient(DOCKER)
	warn(err)
	//  check connection
	err = c.Ping()
	warn(err)
}

// // ////////
//   main  //
// //////////

func main() {

	quit = make(chan struct{})

	flag.BoolVar(&IGNORE_CONFLICT, "ignore", IGNORE_CONFLICT, "ignore conflicts name when creating container")
	flag.IntVar(&N, "n", N, "how many containers to start in parallel")
	flag.IntVar(&B, "b", B, "how many containers to start in on batch")
	flag.StringVar(&NAME, "name", NAME, "name of experiment (measurment and file name)")
	flag.StringVar(&DOCKER, "docker", DOCKER, "docker url")
	flag.StringVar(&INFLUX, "influx", INFLUX, "influx url")
	flag.StringVar(&FILE, "file", FILE, "file to influx data")
	flag.DurationVar(&REPORT, "report", REPORT, "report interval")
	flag.DurationVar(&STORE, "store", STORE, "store interval")

	// parse params
	flag.Parse()

	influx.New()
	initDocker()

	cmds := map[string]func(){
		"events":         storeEvents,
		"rmall":          rmAll,
		"killall":        killAll,
		"printInfo":      printInfo,
		"pull":           pullIMAGE,
		"t1":             t1,
		"tn":             tn,
		"tb":             tb,
		"tnb":            tnb,
		"reportStatuses": reportStatuses,
		"printStatuses":  printStatuses,
		"getall": func() {
			for _, id := range getAllIds(true) {
				println(id)
			}
		},
	}

	// precheck
	for _, cmd := range flag.Args() {
		_, ok := cmds[cmd]
		if !ok {
			log.Fatalf("cmd %q not found", cmd)
		}
	}

	// fire
	wg := sync.WaitGroup{}
	for _, cmd := range flag.Args() {
		wg.Add(1)
		f := func(cmd string) {
			cmds[cmd]()
			wg.Done()
		}
		f(cmd)
	}
	close(quit)
	wg.Wait()
}
