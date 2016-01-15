package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/process"
)

func loadPid(filename string) (int32, error) {

	pidfile, err := os.Open(filename)
	if err != nil {
		return 0, err
	}

	b, err := ioutil.ReadAll(pidfile)
	if err != nil {
		return 0, err
	}
	if err != nil {
		warn(err)
		return 0, err
	}

	pid, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, err
	}
	if err != nil {
		warn(err)
		return 0, err
	}

	alive, err := process.PidExists(int32(pid))
	if err != nil {
		return 0, err
	}

	if !alive {
		return 0, fmt.Errorf("Pid not exists: %v", pid)
	}

	return int32(pid), nil
}

// dockerData gather date from docker daemon directly (using DOCKER_HOST)
// and from proc/DOCKER_PID/status and publish those to conn
func storeProc(pidfile string, interval time.Duration) {
	pid, err := loadPid(pidfile)
	if err != nil {
		warn(err)
		return
	}

	p, err := process.NewProcess(int32(pid))
	warn(err)
	if err != nil {
		return
	}

	// threads
	threads, err := p.NumThreads()
	warn(err)
	if err != nil {
		return
	}

	mi, err := p.MemoryInfo()
	warn(err)
	if err != nil {
		return
	}

	procData := map[string]interface{}{"threads": threads, "vmsize": int(mi.VMS), "rss": int(mi.RSS)}
	log.Println("proc = ", procData)
	store("process", nil, procData)
}
