package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"
)

// go tool -n pprof
const (
	pprofBin    = "./pprof"
	pprofTmpdir = "pprof_tmpdir"
)

var (
	absDir string
)

func initProfiles() {
	wd, _ := os.Getwd()
	absDir = path.Join(wd, pprofTmpdir)
	err := os.MkdirAll(absDir, 0777)
	ok(err)
}

func getProfile(appUrl string, duration time.Duration) {
	getPprof(appUrl, "profile", duration)
}

func getHeap(appUrl string, duration time.Duration) {
	getPprof(appUrl, "heap", duration)
}

func getPprof(appUrl, kind string, duration time.Duration) {

	profileUrl := appUrl + "/debug/pprof/" + kind
	c := exec.Command(pprofBin, "-seconds", strconv.Itoa(int(duration/time.Second)), profileUrl)
	log.Printf("pprof bin: %q", c.Args)
	// c := exec.Command("env")

	c.Env = []string{fmt.Sprintf("PPROF_TMPDIR=%s", absDir)}
	b, err := c.CombinedOutput()
	if !warn(err) {
		log.Printf("pprof: grabed profile output = %s", b)
	}
	log.Println("pprof: done")
}
