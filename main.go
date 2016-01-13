package main

import (
	"flag"
	"log"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

var (
	DOCKER_URL = "http://127.0.0.1:8080"
	// DOCKER_URL     = "unix:///var/run/docker.sock" // panic: [main.create:168] Post http://unix.sock/containers/create?name=tn-1452521422-105: dial unix /var/run/docker.sock: connect: resource temporarily unavailable // unix

	DOCKER_LOG     = "/var/log/docker.log"
	DOCKER_PIDFILE = "/var/run/docker.pid"
	// INFLUX          = "http://127.0.0.1:8086"
	INFLUX          = "file://influx.data"
	N               = 100  // in parallel
	B               = 1000 // how many in one batch
	IGNORE_CONFLICT = true
	NAME            = "docker" // measurment name
	IMAGE           = "alpine"
	CMD             = "sleep 8640000"
	REPORT          = time.Duration(1 * time.Second)
	SLEEP           = time.Duration(1 * time.Second)
	PPROF_SECONDS   = time.Duration(30 * time.Second)

	dockerClient *docker.Client
	wg           sync.WaitGroup
	quit         chan struct{} = make(chan struct{})
)

func initDocker() {
	// connect docker
	// c, err := docker.NewClientFromEnv()
	var err error
	dockerClient, err = docker.NewClient(DOCKER_URL)
	warn(err)
	//  check connection
	err = dockerClient.Ping()
	warn(err)
}

// // ////////
//   main  //
// //////////

func init() {

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
	flag.DurationVar(&REPORT, "report", REPORT, "store interval")
	flag.DurationVar(&SLEEP, "sleep", SLEEP, "sleep interval")
	flag.DurationVar(&PPROF_SECONDS, "pprof", PPROF_SECONDS, "sleep interval")

	flag.Parse()
}

func main() {

	// parse params
	initDocker()

	cmds := map[string]func(){
		// service types
		"sched":    storeSched,
		"events":   storeEvents,
		"proc":     storeProc,
		"statuses": storeStatuses,
		"info":     storeInfo,

		// command (blocking)
		"rmall":     rmAll,
		"killall":   killAll,
		"printInfo": printInfo,
		"pull":      pullIMAGE,
		"t1":        t1,
		"tn":        tn,
		"tb":        tb,
		"tnb":       tnb,
		"getall": func() {
			for _, id := range getAllIds(true) {
				println(id)
			}
		},
		"sleep": func() {
			time.Sleep(SLEEP)
		},
		"pprof": func() {
			go func() {
				for {
					getProfile(DOCKER_URL)
				}
			}()

		},
		"trace": func() {
			go func() {
				for {
					getTrace(DOCKER_URL)
				}
			}()

		},
	}

	// precheck
	for _, cmd := range flag.Args() {
		_, ok := cmds[cmd]
		if !ok {
			l := []string{}
			for k, _ := range cmds {
				l = append(l, k)
			}
			log.Println("available commands:", strings.Join(l, ", "))
			log.Fatalf("cmd %q not found", cmd)
		}
	}

	// fire
	store("logs", nil, map[string]interface{}{"message": "started"})
	for _, cmd := range flag.Args() {
		wg.Add(1)
		f := func(cmd string) {
			cmds[cmd]()
			wg.Done()
		}
		f(cmd)
	}
	store("logs", nil, map[string]interface{}{"message": "done"})
	close(quit)
	wg.Wait()
}
