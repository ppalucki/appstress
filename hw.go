package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
)

import (
	"os/exec"
)

func collectOuput(cmds ...string) string {

	buf := &bytes.Buffer{}

	gather := func(cmd string) {
		// naive shell split function replace with exec.Command ("sh", "-c", cmd tricks) if required
		fields := strings.Fields(cmd)
		c := exec.Command(fields[0], fields[1:len(fields)]...)
		output, err := c.CombinedOutput()
		io.WriteString(buf, fmt.Sprintf("--- %s ---\n", cmd))
		if err != nil {
			msg := fmt.Sprintf("cannot get output from cmd %q err = %q:", cmd, err)
			buf.WriteString(msg)
			log.Printf(msg)
		} else {
			_, err = io.Copy(buf, bytes.NewBuffer(output))
			if err != nil {
				panic(err)
			}
		}
		buf.WriteByte('\n')
	}

	for _, cmd := range cmds {
		gather(cmd)

	}

	return string(buf.Bytes())
}

// hwinfo capture output from:
// lscpu
// free -mt
// du -h
func hwinfo() string {
	return collectOuput(
		"lscpu",
		"free -mt",
		"df -m",
	)
}

// swinfo catpure info about sw stack
func swinfo() string {
	return collectOuput(
		"docker version",
		"docker info",
		"uname -a",
		"cat /etc/os-release",
	)
}

func dumpStack() {
	log.Println(hwinfo())
	log.Println(swinfo())
}

func storeStack() {
	storeLog("hwinfo:", hwinfo())
	storeLog("swinfo:", swinfo())
}
