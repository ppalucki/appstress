package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// docker default location
	DEFAULT_DOCKER_URL = "http://127.0.0.1:8080"
	// DEFAULT_DOCKER_URL     = "unix:///var/run/docker.sock" // panic: [main.create:168] Post http://unix.sock/containers/create?name=tn-1452521422-105: dial unix /var/run/docker.sock: connect: resource temporarily unavailable // unix

	// influx default location
	DEFAULT_INFLUX_FILE = "influx.data"
	DEFAULT_INFLUX_URL  = "file://" + DEFAULT_INFLUX_FILE // current working directory
	// DEFAULT_INFLUX_URL = "http://localhost:8086/write?db=docker"

)

var (
	dockerUrl = flag.String("dockerUrl", DEFAULT_DOCKER_URL, "docker url eg. http://127.0.0.1:8080 or unix:///var/run/docker.sock")

	allOn = flag.Bool("all", false, "all on (the same ass -info -events -proc -sched)")

	// aggregated information about container and general from /info
	infoOn  = flag.Bool("info", false, "gather data from /info API endpoint (from 1.10.x contains aggregated information about no of containers)")
	infoInt = flag.Duration("infoInt", 1*time.Second, "info interval")

	// status aggregated information of /containers API
	statusOn  = flag.Bool("status", false, "gather data from /containers API endpoint and aggregate this data")
	statusInt = flag.Duration("statusInt", 1*time.Second, "status interval")

	// events watch over /events API
	eventsOn = flag.Bool("events", false, "watch on events and pass them to 'evens' influx measuremnt")

	// process stats from procfs /proc/PID/status
	procOn  = flag.Bool("proc", false, "gather data about process from procfs /procs based on PID given with procPID")
	procPid = flag.String("procPid", "/var/run/docker.pid", "location of docker pid file")
	procInt = flag.Duration("procInt", 1*time.Second, "proc data gather interval")

	// sched tracing (base oo GO
	schedOn  = flag.Bool("sched", false, "sched on - gather and expose to scheduler tracing data like: threads, runqueue lenght, gomaxprocs, no of goroutines (http://www.goinggo.net/2015/02/scheduler-tracing-in-go.html, https://golang.org/pkg/runtime/")
	schedLog = flag.String("schedLog", "/var/log/docker.log", "location of docker.log with GODEBUG=schedtrace=10000")

	// profile
	profileOn  = flag.Bool("profile", false, "dump cpu profile data 'go tool pprof' and store it within pprof_tmpdir")
	profileDur = flag.Duration("profileDur", 10*time.Second, "profile duration")
	profileInt = flag.Duration("profileInt", 1*time.Second, "profile interval")

	// trace (uses go tool trace)
	traceOn  = flag.Bool("trace", false, "tracing on")
	traceDur = flag.Duration("traceDur", 10*time.Second, "tracing duration")
	traceInt = flag.Duration("traceInt", 1*time.Second, "tracing interval")

	// one time hw and sw info stack dump
	stackOn = flag.Bool("stack", false, "store & dump stack info")

	// influx
	influxUrl = flag.String("influx", DEFAULT_INFLUX_URL, "where to store influx data: alternative use: 'http://localhost:8086/write?db=docker'")

	// feedInflux
	feedInfluxSrc = flag.String("feedInflux", "", "onetime action that copies data from file to influxUrl")
	influxBatch   = flag.Int("feedLines", 100, "batch size")

	// sleep command
	sleepDuration = flag.Duration("sleep", 1*time.Second, "duration of sleep command")

	// test specific
	N = flag.Int("n", 100, "how many containers(tn) or batches (tnb) to start in parallel")
	B = flag.Int("b", 1000, "how many containers to start in on batch")

	// docker options
	NAME  = flag.String("name", "docker", "name tag - name of experiment")
	IMAGE = flag.String("image", "alpine", "docker image: like 'alpine' or 'jess/stress'")
	CMD   = flag.String("cmd", "sleep 8640000", "docker entrypoint & cmd overwrite: eg. 'watch -n 1 -- stress -c 1 -t 1'")
	TTY   = flag.Bool("tty", false, "allocate tty")
	NET   = flag.String("net", "", " empty or kind of network: eg. 'net', 'none', 'host'")

	// runtime vars
	wg   sync.WaitGroup
	quit = make(chan struct{})
)

// // ////////
//   main  //
// //////////

func sleep() {
	time.Sleep(*sleepDuration)
}

func saveTraces(appUrl string, duration, interval time.Duration) {
}

// loop funcation and handle waitGroup and quit channel
func loop(interval time.Duration, f func()) {
	go func() {
		name := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		wg.Add(1)
		defer func() {
			wg.Done()
		}()
		t := time.NewTicker(interval)
		for {
			select {
			case <-t.C:
				f()
			case _, ok := <-quit:
				log.Printf("f %q got quit %v\n", name, ok)
				return
			}
		}
	}()
}

func main() {

	flag.Parse()

	// just copy influxFile to influxUrl
	if *feedInfluxSrc != "" {
		feedInflux(*feedInfluxSrc, *influxUrl)
		return
	}

	initInflux(*influxUrl)

	if !initDocker(*dockerUrl) {
		log.Printf("cannot connect to docker: %q\n", *dockerUrl)
		return
	}

	all := []*bool{
		infoOn,
		eventsOn,
		procOn,
		schedOn,
		// statusOn - disabled becuase it is possible to get data from docker info about number of containers
	}
	if *allOn {
		for _, v := range all {
			*v = true
		}
	}

	if *infoOn {
		loop(*infoInt, func() {
			storeInfo(*infoInt)
		})
	}

	if *statusOn {
		storeStatuses(*statusInt)
	}

	if *eventsOn {
		storeEvents()
	}

	if *procOn {
		loop(*procInt, func() {
			storeProc(*procPid, *procInt)
		})
	}

	if *schedOn {
		go storeSched(*schedLog)
	}

	if *profileOn {
		initProfiles()
		loop(*profileInt, func() {
			getProfile(*dockerUrl, *profileDur)
			// getHeap(*dockerUrl, *profileDur)
		})
	}

	if *traceOn {
		loop(*traceInt, func() {
			getTrace(*dockerUrl, *traceDur)
		})
	}

	loop := false

	setLoop := func() {
		loop = true
	}

	cmds := map[string]func(){
		// command (blocking)
		"loop":       setLoop,
		"storestack": storeStack,
		"rmall":      rmAll,
		"killall":    killAll,
		"pull":       pullIMAGE,
		"sleep":      sleep,

		// scenarios
		"t1":      t1,
		"tn":      tn,
		"tb":      tb,
		"tnb":     tnb,
		"doublen": doubleN,
		"doubleb": doubleB,

		// debuggers
		"dumpinfo":     dumpInfo,
		"dumpstatuses": dumpStatuses,
		"dumpstack":    dumpStack,
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

	cwd, err := os.Getwd()
	ok(err)

	// fire
	storeLog("started in cwd = ", fmt.Sprintf("%q", cwd))

	tasks := flag.Args()

	for {
		for _, cmd := range tasks {
			wg.Add(1)
			f := func(cmd string) {
				begin := time.Now()
				storeLog("cmd start:", cmd)
				cmds[cmd]()
				duration := time.Since(begin)
				storeLog("cmd done:", cmd, "(", duration.String(), ")")
				wg.Done()
			}
			f(cmd)
		}
		if !loop {
			break
		}
	}
	storeLog("done - quit")
	close(quit)

	wg.Wait()
}
