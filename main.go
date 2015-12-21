package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/ppalucki/dockerstress/influx"
)

var (
	DOCKER          = "http://127.0.0.1:8080"
	N               = 100  // in parallel
	B               = 1000 // how many in one batch
	IGNORE_CONFLICT = true
	NAME            = "docker" // measurment name

	c *docker.Client
)

func ok(err error) {
	if err != nil {
		pc, file, line, ok := runtime.Caller(1)
		fn := runtime.FuncForPC(pc)
		var name string
		if fn != nil {
			name = fn.Name()
		} else {
			name = file
		}
		if ok && false {
			log.Fatalf("ERROR [%s:%d] %s\n", name, line, err)
		}
		panic(err)
	}
}

func connectDocker() {
	// connect docker
	// c, err := docker.NewClientFromEnv()
	var err error
	c, err = docker.NewClient(DOCKER)
	ok(err)
	//  check connection
	err = c.Ping()
	ok(err)
}

// // //////////
//   influx  //
// ////////////

// // ////////
//   main  //
// //////////

func printInfo() {
	m := info()
	mb, err := json.MarshalIndent(m, "", "  ")
	ok(err)
	fmt.Println(string(mb))
}

// run args cmd as functions
func cmds() {

	stop := make(chan struct{})

	var show, influx, file bool

	cmds := map[string]func(){
		"influx": func() {
			influx = true
		},
		"file": func() {
			file = true
		},
		"show": func() {
			show = true
		},
		"events": func() {
			events(show, influx, file)
		},
		"rmall": func() {
			rmAll()
		},
		"killall": func() {
			killAll()
		},
		"pull": func() {
			pull("alpine")

		},
		"info": func() {
			printInfo()
		},
		"t1": func() {
			run("t1", "alpine", "sleep 864000")
		},
		"tn": func() {
			runN(N, "tn", "alpine", "sleep 864000")
		},
		"tb": func() {
			runB(B, "tb", "alpine", "sleep 864000")
		},
		"tnb": func() {
			runBonN(B, N, "tnb", "alpine", "sleep 864000")
		},

		"2tn": func() {
			runN(N, "2tn", "alpine", "sleep 864000")
		},
		"2tb": func() {
			runB(B, "2tb", "alpine", "sleep 864000")
		},
		"2tnb": func() {
			runBonN(B, N, "2tnb", "alpine", "sleep 864000")
		},

		"sleep": func() {
			time.Sleep(5 * time.Second)
		},

		"running": func() {
			log.Println("running:", runnning())
		},
		"statuses": func() {
			statuses := getAllStatuses(true)
			fmt.Printf("statuses = %#v\n", statuses)
		},
		"getall": func() {
			for _, id := range getAllIds(true) {
				println(id)
			}
		},
		"reportrunning": func() {
			go func(stop chan struct{}) {
				ticker := time.NewTicker(1 * time.Second)
				for {
					select {
					case <-ticker.C:
						log.Println("running:", runnning())
					case <-stop:
						break
					}
				}
			}(stop)
		},
		"reportstatuses": func() {
			go func(stop chan struct{}) {
				ticker := time.NewTicker(1 * time.Second)
				for _ = range ticker.C {
					statuses := getAllStatuses(true)
					for k, v := range statuses {
						log.Printf("* %q = %d\n", k, v)
					}
				}
			}(stop)
		},
	}

	flag.BoolVar(&IGNORE_CONFLICT, "ignore", IGNORE_CONFLICT, "ignore conflicts name when creating container")
	flag.IntVar(&N, "n", N, "how many containers to start in parallel")
	flag.IntVar(&B, "b", B, "how many containers to start in on batch")
	flag.StringVar(&NAME, "name", NAME, "name of experiment (measurment and file name)")
	flag.StringVar(&DOCKER, "docker", DOCKER, "docker url")
	help := flag.Bool("h", false, "help")

	// parse params
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		fmt.Println("Available commands:")
		for cmd, _ := range cmds {
			fmt.Println(cmd)
		}
		os.Exit(0)
	}

	// precheck
	for _, cmd := range flag.Args() {
		_, ok := cmds[cmd]
		if !ok {
			log.Fatalf("cmd %q not found", cmd)
		}
	}

	wg := sync.WaitGroup{}
	for _, cmd := range flag.Args() {
		wg.Add(1)
		f := func(cmd string) {
			cmds[cmd]()
			wg.Done()
		}
		f(cmd)
	}
	close(stop)
	wg.Wait()
}

func main() {
	influx.New()
	connectDocker()
	cmds()
}
