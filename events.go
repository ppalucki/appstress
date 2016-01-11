package main

import "github.com/fsouza/go-dockerclient"

// start an goroutine that collects events in influx store
func storeEvents() {
	listener := make(chan *docker.APIEvents)
	err := dockerClient.AddEventListener(listener)
	ok(err)
	go func() {
		for {
			e := <-listener
			store("events", map[string]string{}, map[string]interface{}{"value": 1, "kind": e.Status})
		}
	}()
}
