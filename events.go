package main

import (
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/ppalucki/dockerstress/influx"
)

// start an goroutine that collects events in influx store
func events() {

	listener := make(chan *docker.APIEvents)
	err := c.AddEventListener(listener)
	ok(err)
	go func() {
		for {
			e := <-listener
			influx.Store("events", map[string]string{"type": e.Status}, map[string]interface{}{"value": 1}, time.Now())
		}
	}()
}
