package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/process"
)

func loadPid() (int32, error) {

	pidfile, err := os.Open(DOCKER_PIDFILE)
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
func storeProc() {
	go func() {
		for {
			pid, err := loadPid()
			if err != nil {
				warn(err)
				time.Sleep(REPORT)
				continue
			}

			p, err := process.NewProcess(int32(pid))
			warn(err)
			if err != nil {
				time.Sleep(REPORT)
				continue
			}

			// threads
			threads, err := p.NumThreads()
			warn(err)
			if err != nil {
				time.Sleep(REPORT)
				continue
			}

			mi, err := p.MemoryInfo()
			warn(err)
			if err != nil {
				time.Sleep(REPORT)
				continue
			}

			store("process", nil, map[string]interface{}{"threads": threads, "vmsize": int(mi.VMS), "rss": int(mi.RSS)})

			time.Sleep(REPORT)
		}
	}()
}
