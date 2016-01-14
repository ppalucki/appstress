package main

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

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
