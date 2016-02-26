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
	INFLUX_FILE = "influx.data"
	INFLUX_URL  = "file://" + INFLUX_FILE
	// INFLUX_URL = "http://localhost:8086/write?db=docker"
	DEFAULT_DOCKER_URL = "http://127.0.0.1:8080"
	// DEFAULT_DOCKER_URL     = "unix:///var/run/docker.sock" // panic: [main.create:168] Post http://unix.sock/containers/create?name=tn-1452521422-105: dial unix /var/run/docker.sock: connect: resource temporarily unavailable // unix
)

var (
	// dockerUrl = flag.String("dockerUrl", "http://127.0.0.1:8080", "docker url")
	dockerUrl = flag.String("dockerUrl", DEFAULT_DOCKER_URL, "docker url eg. http://127.0.0.1:8080")

	allOn = flag.Bool("all", false, "all on (the same ass -info -events -proc -sched)")

	infoOn  = flag.Bool("info", false, "info on")
	infoInt = flag.Duration("infoInt", 1*time.Second, "status interval")

	statusOn  = flag.Bool("status", false, "status on")
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
	influxUrl = flag.String("influx", INFLUX_URL, "where to store influx data")

	stackOn = flag.Bool("stack", false, "store & dump stack info")

	// feedInflux
	feedInfluxSrc = flag.String("feedInflux", "", "onetime action that copies data from file to influxUrl")
	influxBatch   = flag.Int("feedLines", 100, "batch size")

	sleepDuration = flag.Duration("sleep", 1*time.Second, "duration of sleep")

	// test specific
	N = flag.Int("n", 100, "how many containers(tn) or batches (tnb) to start in parallel")
	B = flag.Int("b", 1000, "how many containers to start in on batch")

	// docker options
	NAME  = flag.String("name", "docker", "name tag - name of experiment")
	IMAGE = flag.String("image", "alpine", "docker image")
	CMD   = flag.String("cmd", "sleep 8640000", "docker entrypoint & cmd overwrite")
	TTY   = flag.Bool("tty", false, "allocate tty")
	NET   = flag.String("net", "", " empty or kind of network")

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
		// statusOn,
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
