package main

import (
	"flag"
	"log"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

const (
	INFLUX_URL = "file://influx.data"
	// INFLUX_URL = "http://localhost:8086/write?db=docker"
	DOCKER_URL = "http://127.0.0.1:8080"
	// DOCKER_URL     = "unix:///var/run/docker.sock" // panic: [main.create:168] Post http://unix.sock/containers/create?name=tn-1452521422-105: dial unix /var/run/docker.sock: connect: resource temporarily unavailable // unix
)

var (
	dockerUrl = flag.String("dockerUrl", DOCKER_URL, "docker url")

	infoOn  = flag.Bool("info", true, "info on")
	infoInt = flag.Duration("infoInt", 1*time.Second, "status interval")

	statusOn  = flag.Bool("status", true, "status on")
	statusInt = flag.Duration("statusInt", 1*time.Second, "status interval")

	eventsOn = flag.Bool("events", false, "event on")

	procOn  = flag.Bool("proc", false, "proc on")
	procPid = flag.String("procPid", "/var/run/docker.pid", "location of docker pid file")
	procInt = flag.Duration("procInt", 1*time.Second, "proc interval")

	schedOn  = flag.Bool("sched", false, "sched on")
	schedLog = flag.String("schedLog", "/var/log/docker.log", "location of docker.log")

	profileOn  = flag.Bool("profile", false, "profile on")
	profileDur = flag.Duration("profileDur", 10*time.Second, "profile duration")
	profileInt = flag.Duration("profileInt", 1*time.Second, "profile interval")

	traceOn   = flag.Bool("trace", false, "tracing on")
	traceDur  = flag.Duration("traceDur", 10*time.Second, "tracing duration")
	traceInt  = flag.Duration("traceInt", 1*time.Second, "tracing interval")
	influxUrl = flag.String("influx", "file://influx.data", "where to store influx data")

	// test specific
	N = flag.Int("n", 100, "how many containers(tn) or batches (tnb) to start in parallel")
	B = flag.Int("b", 1000, "how many containers to start in on batch")

	NAME  = flag.String("name", "docker", "name tag")
	IMAGE = flag.String("image", "alpine", "docker image")
	CMD   = flag.String("cmd", "sleep 8640000", "docker cmd")

	// runtime vars
	dockerClient *docker.Client
	wg           sync.WaitGroup
	quit         chan struct{} = make(chan struct{})
)

// // ////////
//   main  //
// //////////

func init() {
	flag.Parse()

	initDocker()
	initInflux(*influxUrl)

	if *infoOn {
		storeInfo(*infoInt)
	}

	if *statusOn {
		storeStatuses(*statusInt)
	}

	if *eventsOn {
		storeEvents()
	}

	if *procOn {
		storeProc(*procPid, *procInt)
	}

	if *schedOn {
		sched(nil, *schedLog)
	}

	if *profileOn {
		storeProfile(*dockerUrl, *profileDur, *profileInt)
	}
}

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

func sleep() {
	time.Sleep(time.Second)
}

func main() {

	// parse params
	initDocker()

	cmds := map[string]func(){
		// command (blocking)
		"rmall":   rmAll,
		"killall": killAll,
		"pull":    pullIMAGE,
		"t1":      t1,
		"tn":      tn,
		"tb":      tb,
		"tnb":     tnb,
		"sleep":   sleep,
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
