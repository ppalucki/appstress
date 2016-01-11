package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

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

// according func (s *State) String() string
// possible states are "up", "restarting", "removal", "dead", "created", "exited")
// we map this to StateString
func statuses(all bool) map[string]int {
	s := make(map[string]int)
	for _, c := range getAll(all) {
		var state string
		switch {
		case strings.Contains(c.Status, "(Paused)"):
			state = "paused"
		case strings.Contains(c.Status, "Restarting"):
			state = "restarting"
		default:
			state = strings.ToLower(strings.SplitN(c.Status, " ", 1)[0])
		}
		s[state] += 1
	}
	return s
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
	if err != nil {
		warn(err)
		return nil
	}
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
