package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

var (
	DOCKER          = "http://127.0.0.1:8080"
	N               = 100  // in parallel
	B               = 1000 // how many in one batch
	IGNORE_CONFLICT = true
	INFLUX_URL      = "http://localhost:8086/write?db=docker" // dbname
	NAME            = "docker"                                // measurment name
	c               *docker.Client
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

// // //////////
//   docker  //
// ////////////

func getAll(all bool) []docker.APIContainers {
	opts := docker.ListContainersOptions{All: all}
	containers, err := c.ListContainers(opts)
	ok(err)
	return containers
}

func getAllStatuses(all bool) (statuses map[string]int) {
	statuses = make(map[string]int)
	for _, c := range getAll(all) {
		statuses[c.Status] += 1
	}
	return
}

func getAllIds(all bool) []string {
	containers := getAll(all)
	ids := []string{}
	for _, con := range containers {
		ids = append(ids, con.ID)
	}
	return ids
}

func rmAll() {
	ids := getAllIds(true)
	for _, id := range ids {
		rm(id)
	}
}

func killAll() {
	ids := getAllIds(false)
	for _, id := range ids {
		kill(id)
	}
}

func kill(id string) {
	err := c.KillContainer(docker.KillContainerOptions{id, docker.SIGKILL})
	ok(err)
}

func rm(id string) {

	opts := docker.RemoveContainerOptions{
		Force: true,
		ID:    id,
	}
	err := c.RemoveContainer(opts)
	ok(err)
}

// b number of containers in batch running in n goroutines
func runBonN(b, n int, baseName, image, cmd string) {
	wg := sync.WaitGroup{}
	wg.Add(n) // number of goroutines
	for i := 0; i < n; i++ {
		go func(i int) {
			for j := 0; j < b; j++ {
				name := fmt.Sprintf("%s-%d-%d", baseName, i, j)
				run(name, image, cmd)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func runB(b int, baseName, image, cmd string) {
	for i := 0; i < b; i++ {
		name := fmt.Sprintf("%s-%d", baseName, i)
		run(name, image, cmd)
	}
}

func runN(n int, baseName, image, cmd string) {

	wg := sync.WaitGroup{}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			name := fmt.Sprintf("%s-%d", baseName, i)
			run(name, image, cmd)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func runnning() int {
	m := info()
	v, err := strconv.Atoi(m["Containers"])
	ok(err)
	return v
}

func info() map[string]string {
	info, err := c.Info()
	ok(err)
	m := info.Map()
	return m
}

func pull(name string) string {
	// get or create an image
	i, err := c.InspectImage(name)
	switch err {
	case docker.ErrNoSuchImage:
		// pull stress image
		err = c.PullImage(docker.PullImageOptions{
			Repository: "alpine",
			Tag:        "latest",
		}, docker.AuthConfiguration{})
		ok(err)
		i, err = c.InspectImage("alpine")
		ok(err)
	default:
		ok(err)
	}
	// log.Printf("using image %q = %v\n", name, i.ID)
	return i.ID

}

// run returns container.ID
func create(name, image, cmd string) string {
	cmds := strings.Split(cmd, " ")
	config := &docker.Config{Cmd: cmds, Image: image, NetworkDisabled: true}
	cc := docker.CreateContainerOptions{Name: name, Config: config}
	cont, err := c.CreateContainer(cc)
	if IGNORE_CONFLICT && err == docker.ErrContainerAlreadyExists {
		log.Println("create ignored - already exists!")
		return ""
	} else {
		ok(err)
	}
	return cont.ID
}

func start(id string) {
	hc := &docker.HostConfig{}
	err := c.StartContainer(id, hc)
	ok(err)

}

func run(name, image, cmd string) string {
	id := create(name, image, cmd)
	start(id)
	return id

}

type stats struct {
	sync.RWMutex
	m       map[string]int
	running int
	when    time.Time
}

func newStats(running int) *stats {
	s := &stats{}
	s.m = make(map[string]int)
	s.running = running
	s.when = time.Now()
	return s
}

func (s *stats) add(name string) {
	s.Lock()
	s.m[name] += 1

	// update internal counters accoring to state graph
	switch name {
	case "die":
		s.running -= 1
	case "start":
		s.running += 1
	}
	s.when = time.Now()
	s.Unlock()
}

func (s *stats) dec(name string) {
	s.Lock()
	s.m[name] -= 1
	s.when = time.Now()
	s.Unlock()
}

func (s *stats) show() {
	s.RLock()
	defer s.RUnlock()
	var b bytes.Buffer

	if len(s.m) > 0 {
		// sort keys
		keys := []string{}
		for key, _ := range s.m {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			fmt.Fprintf(&b, "%s=%d,", key, s.m[key])
		}
		log.Println(s.running, b.String())
	}
}

// -------------------------------------------------- influx

// dumpInflux date in influx 9 format
// measurement[,tag_key1=tag_value1...] field_key=field_value[,field_key2=field_value2] [timestamp]
// eg. measurement,tkey1=tval1,tkey2=tval2 fkey=fval,fkey2=fval2 1234567890000000000
// measurement
func (s *stats) dumpInflux(measurement string, tags map[string]string) (b *bytes.Buffer) {

	b = new(bytes.Buffer)

	s.RLock()
	defer s.RUnlock()
	b.WriteString(measurement)
	if len(tags) > 0 {
		for k, v := range tags {
			fmt.Fprintf(b, ",%s=%s", k, v)
		}
	}
	b.WriteByte(' ')
	if len(s.m) > 0 {
		for k, v := range s.m {
			fmt.Fprintf(b, "%s=%d,", k, v)
		}
	}
	fmt.Fprintf(b, "running=%v ", s.running)
	fmt.Fprintf(b, "%d", s.when.UnixNano())

	return
}

type influx struct {
	// measrument to Buffer
	ms map[string]bytes.Buffer
}

func newInflux() *influx {
	i := new(influx)
	i.ms = make(map[string]bytes.Buffer)
	return i
}

func (i *influx) log(measurement string, tags map[string]string, data [string]int) {
	b := i.ms[measurement]
	b.WriteString(measurement)
	if len(tags) > 0 {
		for k, v := range tags {
			fmt.Fprintf(b, ",%s=%s", k, v)
		}
	}
	b.WriteByte(' ')
	if len(s.m) > 0 {
		for k, v := range s.m {
			fmt.Fprintf(b, "%s=%d,", k, v)
		}
	}
	fmt.Fprintf(b, "running=%v ", s.running)
	fmt.Fprintf(b, "%d", s.when.UnixNano())

}

func (i *inlux) feedInflux() {
	data := s.dumpInflux(NAME, nil)
	resp, err := http.Post(INFLUX_URL, "application/octet-stream", data)
	if err != nil {
		log.Print("err problem with connecting to influx:", err)
	} else {
		if resp.StatusCode != 200 {
			txt, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			log.Print("err problem with posting data to influx:", string(txt))
		}
	}
}

func (i *stats) feedFile(w io.Writer) {
	data := s.dumpInflux("docker", nil)
	data.WriteByte('\n')
	n, err := io.Copy(w, data)
	if err != nil {
		panic(err)
	}
	log.Printf("%d bytes written\n", n)
}

// start an goroutine and print all events
func events(show, influx, file bool) {
	running := len(getAllIds(false))
	s := newStats(running)
	listener := make(chan *docker.APIEvents)
	err := c.AddEventListener(listener)
	ok(err)
	go func() {
		for {
			select {
			case e := <-listener:
				s.add(e.Status)
			case <-time.After(1 * time.Second):
				log.Println("no events observed")
			}
		}
	}()

	ticker := time.NewTicker(1 * time.Second)

	var (
		f *os.File
	)
	if file {
		f, err = os.Create("influx.data")
		if err != nil {
			panic(err)
		}
	}

	// just s
	go func() {
		for _ = range ticker.C {
			if show {
				s.show()
			}
			if influx {
				s.feedInflux()
			}
			if file {
				s.feedFile(f)
			}

		}
	}()

}

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
	connectDocker()
	cmds()
}
