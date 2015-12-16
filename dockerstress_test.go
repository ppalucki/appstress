package main

import (
	"fmt"
	"testing"
)

func TestInfluxDump(t *testing.T) {
	s := newStats(0)
	s.add("foo")
	d := s.dumpInflux("m", map[string]string{"tag1": "tag1v"})
	fmt.Printf("%q", d)
}
