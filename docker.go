package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	// TODO:
	// docker "github.com/docker/engine-api/client"
	// dockerTypes "github.com/docker/engine-api/types"
)

var (
	dockerClient *docker.Client
)

////////////
//  init  //
////////////

func initDocker(dockerUrl string) bool {
	// connect docker
	// c, err := docker.NewClientFromEnv()
	var err error
	dockerClient, err = docker.NewClient(dockerUrl)
	if warn(err) {
		return false
	}
	//  check connection
	err = dockerClient.Ping()
	return !warn(err)
}

/////////////////////////////
//   low-level docker api  //
/////////////////////////////
func getAll(all bool) []docker.APIContainers {
	opts := docker.ListContainersOptions{All: all}
	containers, err := dockerClient.ListContainers(opts)
	if warn(err) {
		return nil
	}
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
			Repository: name,
			Tag:        "latest",
		}, docker.AuthConfiguration{})
		ok(err)
		i, err = dockerClient.InspectImage(name)
		ok(err)
	default:
		warn(err)
		return ""
	}
	// log.Printf("using image %q = %v\n", name, i.ID)
	return i.ID

}

// run returns container.ID
func create(name string) string {
	cmds := strings.Split(*CMD, " ")
	config := &docker.Config{
		Image: *IMAGE,
		// run options
		Cmd:        cmds[1:len(cmds)],
		Entrypoint: cmds[0:1],
		// stdin/out options
		AttachStderr: false,
		AttachStdin:  false,
		AttachStdout: false,
		Tty:          *TTY,
		StdinOnce:    false,

		// network options
		NetworkDisabled: *NET == "",
	}

	hostConfig := &docker.HostConfig{
		NetworkMode: *NET,
	}

	cc := docker.CreateContainerOptions{Name: name, Config: config, HostConfig: hostConfig}
	cont, err := dockerClient.CreateContainer(cc)
	if err == docker.ErrContainerAlreadyExists {
		log.Println("create ignored - already exists!")
		return ""
	}
	if warn(err) {
		return ""
	}
	return cont.ID
}

func start(id string) bool {
	hc := &docker.HostConfig{}
	err := dockerClient.StartContainer(id, hc)
	return !warn(err)
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

// func runnning() int {
// 	m := info()
// 	v, err := strconv.Atoi(m["ContainersRunning"])
// 	ok(err)
// 	return v
// }

func run(name string) int {
	id := create(name)
	if id != "" {
		if start(id) {
			return 1
		} else {
			return 0
		}
	}
	return 1
}

/////////////////
//  scenarios  //
/////////////////

func timeTaken(st time.Time, cnt int) string {
	dur := float32(time.Since(st)) / float32(time.Second)
	return fmt.Sprintf("duration=%0.2fs containers/s=%0.2f", dur, float32(cnt)/dur)
}

// b number of containers in batch running in n goroutines
func runBonN(b, n int, baseName string) int {
	st := time.Now()
	cnt := 0
	cntch := make(chan int, n*b)
	for i := 0; i < n; i++ {
		go func(i int) {
			for j := 0; j < b; j++ {
				name := fmt.Sprintf("%s-%d-%d", baseName, i, j)
				cntch <- run(name)
			}
		}(i)
	}
	for i := 0; i < n*b; i++ {
		cnt += <-cntch
	}
	storeLog("runBonN done success =", strconv.Itoa(cnt), timeTaken(st, cnt))
	return cnt
}

// run on by one up to B
func runB(b int, baseName string) int {
	st := time.Now()
	cnt := 0
	for i := 0; i < b; i++ {
		name := fmt.Sprintf("%s-%d", baseName, i)
		cnt += run(name)
	}
	storeLog("runB done success=", strconv.Itoa(cnt), timeTaken(st, cnt))
	return cnt
}

// run N containers in parallel
func runN(n int, baseName string) int {
	st := time.Now()
	cnt := 0
	cntch := make(chan int, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			name := fmt.Sprintf("%s-%d", baseName, i)
			cntch <- run(name)
		}(i)
	}
	for i := 0; i < n; i++ {
		cnt += <-cntch
	}
	storeLog("runN done success=", strconv.Itoa(cnt), timeTaken(st, cnt))
	return cnt
}

///////////
//  cmds //
///////////

func pullIMAGE() {
	pull(*IMAGE)
}

func t1() {
	name := fmt.Sprintf("t1-%d", time.Now().Unix())
	run(name)
}
func tn() {
	name := fmt.Sprintf("tn-%d", time.Now().Unix())
	runN(*N, name)
}
func tb() {
	name := fmt.Sprintf("tb-%d", time.Now().Unix())
	runB(*B, name)
}
func tnb() {
	name := fmt.Sprintf("tnb-%d", time.Now().Unix())
	runBonN(*B, *N, name)
}

// log2 increase a number of containers starting from 1 up to n in batch
// with rmall and sleep between
// N defines parallelism
// B - max defines number of containers in batch
func doubleB() {
	b := 1
	for b <= *B {
		name := fmt.Sprintf("doubleB-tnb-%d", time.Now().Unix())
		storeLog(fmt.Sprintf("dobuleB with b=%d (n=%d)", b, *N))
		runBonN(b, *N, name)
		sleep()
		storeLog("rmall")
		rmAll()
		sleep()
		b *= 2
	}
}

// log2 increase a number of containers starting from 1 up to n in N parallel
// with rmall and sleep between
func doubleN() {
	n := 1
	for n <= *N {
		name := fmt.Sprintf("doubleN-tnb-%d", time.Now().Unix())
		storeLog(fmt.Sprintf("dobule with n=%d (b=%d)", n, *B))
		runBonN(*B, n, name)
		sleep()
		storeLog("rmall")
		rmAll()
		sleep()
		n *= 2
	}
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
			store("events", map[string]string{"kind": e.Status}, map[string]interface{}{"value": 1})
		}
	}()
}

// store info response in cyclic manner
func storeInfo(interval time.Duration) {
	start := time.Now()
	i := info()
	duration := time.Since(start)

	cast := func(field string) int {
		v, exists := i[field]
		if !exists {
			panic("wrong field:" + field)
			return 0
		}
		i, err := strconv.Atoi(v)
		ok(err)
		return i
	}

	if i != nil {
		d := map[string]interface{}{
			"containers":        cast("Containers"),
			"containersRunning": cast("ContainersRunning"),
			"containersStopped": cast("ContainersStopped"),
			"containersPaused":  cast("ContainersPaused"),
			"ngoroutines":       cast("NGoroutines"),
			"nfd":               cast("NFd"),
			"duration":          int64(duration / time.Millisecond),
		}
		// log.Printf("info = %q\n", d)
		dump(d)
		store("info", nil, d)
	}
}

// store containers statues in cyclic manner
func storeStatuses(interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		for range t.C {

			start := time.Now()
			s := statuses(true)
			duration := time.Since(start)
			fields := statusesToFields(s)
			fields["duration"] = int64(duration)
			store("statuses", nil, fields)
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
