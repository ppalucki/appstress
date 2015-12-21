package main

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
)

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
