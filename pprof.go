package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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

func init() {
	wd, _ := os.Getwd()
	absDir = path.Join(wd, pprofTmpdir)
	err := os.MkdirAll(absDir, 0777)
	ok(err)
}
func getHeap(appUrl string) {
	getPprof(appUrl, "heap")
}

func getProfile(appUrl string) {
	getPprof(appUrl, "profile")
}

func getTrace(appUrl string) {
	resp, err := http.Get(appUrl + "/debug/pprof/trace?seconds=" + strconv.Itoa(int(PPROF_SECONDS.Seconds())))
	if warn(err) && resp != nil {
		log.Println("trace download error:")
	}
	f, err := ioutil.TempFile(absDir, "trace-"+NAME+"-"+time.Now().Format(time.RFC3339)+"-")
	ok(err)
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	ok(err)
	log.Println("trace written:", n, f.Name())
}

func getPprof(appUrl, kind string) {

	profileUrl := appUrl + "/debug/pprof/" + kind
	c := exec.Command(pprofBin, "-seconds", strconv.Itoa(int(PPROF_SECONDS.Seconds())), profileUrl)
	// c := exec.Command("env")

	c.Env = []string{fmt.Sprintf("PPROF_TMPDIR=%s", absDir)}
	b, err := c.CombinedOutput()
	if !warn(err) {
		fmt.Printf("b = %s\n", b)
	}
	log.Println("Done!")

}
