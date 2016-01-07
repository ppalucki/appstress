package main

import (
	"flag"
	"log"
	"net/url"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/ppalucki/dockerstress/influx"
)

var (
	DOCKER_URL     = "http://127.0.0.1:8080"
	DOCKER_LOG     = "/var/log/docker.log"
	DOCKER_PIDFILE = "docker.pid"
	// INFLUX          = "http://127.0.0.1:8086"
	INFLUX          = "file://influx.data"
	N               = 100  // in parallel
	B               = 1000 // how many in one batch
	IGNORE_CONFLICT = true
	NAME            = "docker" // measurment name
	IMAGE           = "alpine"
	CMD             = "sleep 8640000"
	REPORT          = time.Duration(1 * time.Second)
	STORE           = time.Duration(1 * time.Second)
	SLEEP           = time.Duration(1 * time.Second)

	c    *docker.Client
	quit chan struct{}
)

func initDocker() {
	// connect docker
	// c, err := docker.NewClientFromEnv()
	var err error
	c, err = docker.NewClient(DOCKER_URL)
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

	// test specific
	flag.BoolVar(&IGNORE_CONFLICT, "ignore", IGNORE_CONFLICT, "ignore conflicts name when creating container")
	flag.IntVar(&N, "n", N, "how many containers to start in parallel")
	flag.IntVar(&B, "b", B, "how many containers to start in on batch")

	// docker locations/pid/logs
	flag.StringVar(&DOCKER_URL, "docker_url", DOCKER_URL, "docker url")
	flag.StringVar(&DOCKER_LOG, "docker_log", DOCKER_LOG, "docker log file path with schedule details")
	flag.StringVar(&DOCKER_PIDFILE, "docker_pidfile", DOCKER_PIDFILE, "docker pid file")

	// influx db name and location (file/http)
	flag.StringVar(&NAME, "name", NAME, "name of experiment (measurment and file name)")
	flag.StringVar(&INFLUX, "influx", INFLUX, "influx url")

	// intervals
	flag.DurationVar(&REPORT, "report", REPORT, "report interval")
	flag.DurationVar(&STORE, "store", STORE, "store interval")
	flag.DurationVar(&SLEEP, "sleep", SLEEP, "sleep interval")

	// parse params
	flag.Parse()

	influx.New()
	initDocker()

	cmds := map[string]func(){
		"sched":          storeSched,
		"events":         storeEvents,
		"proc":           storeProc,
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
		"sleep": func() {
			time.Sleep(SLEEP)
		},
		"save": func() {
			u, err := url.Parse(INFLUX)
			ok(err)
			if u.Scheme == "file" {
				err := influx.SaveFile(u.Host)
				ok(err)
			} else {
				err := influx.SaveInflux(INFLUX, "docker")
				ok(err)
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
