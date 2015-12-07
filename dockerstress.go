package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

import docker "github.com/fsouza/go-dockerclient"

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

var (
	c *docker.Client
)

func init() {

	// connect docker
	// c, err := docker.NewClientFromEnv()
	var err error
	c, err = docker.NewClient("http://127.0.0.1:8080")
	ok(err)

	//  check connection
	err = c.Ping()
	ok(err)

}

// // //////////
//   docker  //
// ////////////

func getAll(all bool) []string {
	opts := docker.ListContainersOptions{All: all}
	containers, err := c.ListContainers(opts)
	ok(err)
	ids := []string{}
	for _, con := range containers {
		ids = append(ids, con.ID)
	}
	return ids
}

func rmAll() {
	ids := getAll(true)
	for _, id := range ids {
		rm(id)
	}
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

func cnt() int {
	info, err := c.Info()
	ok(err)
	m := info.Map()
	v, err := strconv.Atoi(m["Containers"])
	ok(err)
	return v
}

func pull(name string) {
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
	log.Printf("using image %q = %v\n", name, i.ID)

}

// run returns container.ID
func create(name, image, cmd string) string {
	cmds := strings.Split(cmd, " ")
	config := &docker.Config{Cmd: cmds, Image: image, NetworkDisabled: true}
	cc := docker.CreateContainerOptions{Name: name, Config: config}
	cont, err := c.CreateContainer(cc)
	ok(err)
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
	m map[string]int
}

func newStats() *stats {
	s := &stats{}
	s.m = make(map[string]int)
	return s
}

func (s *stats) add(name string) {
	s.Lock()
	s.m[name] += 1
	s.Unlock()
}

func (s *stats) dec(name string) {
	s.Lock()
	s.m[name] -= 1
	s.Unlock()
}

func (s *stats) show() {
	s.RLock()
	var b bytes.Buffer

	for k, v := range s.m {
		fmt.Fprintf(&b, "%s=%d ", k, v)
		log.Println(b.String())
	}
	s.RUnlock()
}

// start an goroutine and print all events
func events() {
	s := newStats()
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
	// just s
	go func() {
		for _ = range ticker.C {
			s.show()
		}
	}()

}

// // ////////
//   main  //
// //////////

func main() {
	// flags
	rmAllFlag := flag.Bool("rmall", false, "kurwa")
	flag.Parse()
	events()

	if *rmAllFlag {
		log.Println("rmall")
		rmAll()
	}

	rmAll()

	pull("alpine")

	run("t1", "alpine", "sleep 864000")
	runN(10, "t2", "alpine", "sleep 864000")
	runB(10, "t3", "alpine", "sleep 864000")
	runBonN(10, 10, "t4", "alpine", "sleep 864000")

	// fmt.Printf("cnt = %+v\n", cnt())

	time.Sleep(5 * time.Second)

}
