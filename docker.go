package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
)

var (
	dockerClient *docker.Client
)

////////////
//  init  //
////////////

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

/////////////////////////////
//   low-level docker api  //
/////////////////////////////
func getAll(all bool) []docker.APIContainers {
	opts := docker.ListContainersOptions{All: all}
	containers, err := dockerClient.ListContainers(opts)
	ok(err)
	return containers
}

func info() map[string]string {
	info, err := dockerClient.Info()
	if err != nil {
		warn(err)
		return nil
	}
	m := info.Map()
	return m
}

func kill(id string) {
	err := dockerClient.KillContainer(docker.KillContainerOptions{id, docker.SIGKILL})
	ok(err)
}

func rm(id string) {
	opts := docker.RemoveContainerOptions{
		Force: true,
		ID:    id,
	}
	err := dockerClient.RemoveContainer(opts)
	warn(err)
}

func pull(name string) string {
	// get or create an image
	i, err := dockerClient.InspectImage(name)
	switch err {
	case docker.ErrNoSuchImage:
		// pull stress image
		err = dockerClient.PullImage(docker.PullImageOptions{
			Repository: "alpine",
			Tag:        "latest",
		}, docker.AuthConfiguration{})
		ok(err)
		i, err = dockerClient.InspectImage("alpine")
		ok(err)
	default:
		warn(err)
		return ""
	}
	// log.Printf("using image %q = %v\n", name, i.ID)
	return i.ID

}

// run returns container.ID
func create(name, image, cmd string) string {
	cmds := strings.Split(cmd, " ")
	config := &docker.Config{Cmd: cmds, Image: image, NetworkDisabled: true}
	cc := docker.CreateContainerOptions{Name: name, Config: config}
	cont, err := dockerClient.CreateContainer(cc)
	if err == docker.ErrContainerAlreadyExists {
		log.Println("create ignored - already exists!")
		return ""
	} else {
		ok(err)
	}
	return cont.ID
}

func start(id string) {
	hc := &docker.HostConfig{}
	err := dockerClient.StartContainer(id, hc)
	ok(err)
}

/////////////////////////////
//  high level docker api  //
/////////////////////////////

// according func (s *State) String() string
// possible states are "up", "restarting", "removal", "dead", "created", "exited")
// we map this to StateString
func statuses(all bool) map[string]int {
	s := map[string]int{
		"paused":     0,
		"restarting": 0,
		"up":         0,
		"removal":    0,
		"dead":       0,
		"created":    0,
		"exited":     0,
	}
	for _, c := range getAll(all) {
		var state string
		switch {
		case strings.Contains(c.Status, "(Paused)"):
			state = "paused"
		case strings.Contains(c.Status, "Restarting"):
			state = "restarting"
		default:
			state = strings.ToLower(strings.SplitN(c.Status, " ", 2)[0])
		}
		s[state] += 1
	}
	return s
}

func statusesToFields(s map[string]int) map[string]interface{} {
	fields := map[string]interface{}{}
	for k, v := range s {
		fields[k] = v
	}
	return fields
}

// func statusesToFields(s map[string]int) map[string]interface{} {
//
// 	data := map[string]int{}
// 	for k, v := range s {
// 		k = strings.Split(k, " ")[0]
// 		prev, ok := data[k]
// 		if !ok {
// 			prev = 0
// 		}
// 		data[k] = prev + v
// 	}
//
// 	fields := map[string]interface{}{}
// 	for k, v := range data {
// 		fields[k] = v
// 	}
// 	if len(fields) == 0 {
// 		fields["up"] = 0
// 		fields["exited"] = 0
// 	}
// 	return fields
// }

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

func runnning() int {
	m := info()
	v, err := strconv.Atoi(m["Containers"])
	ok(err)
	return v
}

func run(name, image, cmd string) string {
	id := create(name, image, cmd)
	start(id)
	return id

}

/////////////////
//  scenarios  //
/////////////////

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

// run on by one up to B
func runB(b int, baseName, image, cmd string) {
	for i := 0; i < b; i++ {
		name := fmt.Sprintf("%s-%d", baseName, i)
		run(name, image, cmd)
	}
}

// run N containers in parallel
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

///////////
//  cmds //
///////////

func pullIMAGE() {
	pull(*IMAGE)
}

func t1() {
	name := fmt.Sprintf("t1-%d", time.Now().Unix())
	run(name, *IMAGE, *CMD)
}
func tn() {
	name := fmt.Sprintf("tn-%d", time.Now().Unix())
	runN(*N, name, *IMAGE, *CMD)
}
func tb() {
	name := fmt.Sprintf("tb-%d", time.Now().Unix())
	runB(*B, name, *IMAGE, *CMD)
}
func tnb() {
	name := fmt.Sprintf("tnb-%d", time.Now().Unix())
	runBonN(*B, *N, name, *IMAGE, *CMD)
}

/////////////////////////////
//  monitoring goroutines  //
/////////////////////////////

// start an goroutine that collects events in influx store
func storeEvents() {
	listener := make(chan *docker.APIEvents)
	err := dockerClient.AddEventListener(listener)
	ok(err)
	go func() {
		for {
			e := <-listener
			store("events", nil, map[string]interface{}{"value": 1, "kind": e.Status})
		}
	}()
}

// store info response in cyclic manner
func storeInfo(interval time.Duration) {
	i := info()
	if i != nil {

		containers, err := strconv.Atoi(i["Containers"])
		ok(err)
		goroutines, err := strconv.Atoi(i["NGoroutines"])
		ok(err)
		nfd, err := strconv.Atoi(i["NFd"])
		ok(err)

		d := map[string]interface{}{
			"containers":  containers,
			"ngoroutines": goroutines,
			"nfd":         nfd,
		}
		log.Println("info = ", d)
		store("info", nil, d)
	}
}

// store containers statues in cyclic manner
func storeStatuses(interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		for range t.C {
			s := statuses(true)
			store("statuses", nil, statusesToFields(s))
			log.Println("statuses = ", s)
		}
	}()
}

///////////////////////
//  helper/debuging  //
///////////////////////

func dump(i interface{}) {
	mb, err := json.MarshalIndent(i, "", "  ")
	ok(err)
	log.Println(string(mb))
}

func dumpInfo() {
	dump(info())
}
func dumpStatuses() {
	dump(statuses(true))
}
