package main

import (
	"fmt"
	"log"
	"time"
)

func printStatuses() {
	s := statuses(true)
	log.Printf("statuses = %#v\n", s)
}

func reportStatuses() {
	go func() {
		printStatuses()
		time.Sleep(REPORT * time.Second)
	}()

}

func pullIMAGE() {
	pull(IMAGE)
}

func t1() {
	name := fmt.Sprintf("t1-%d", time.Now().Unix())
	run(name, IMAGE, CMD)
}
func tn() {
	name := fmt.Sprintf("tn-%d", time.Now().Unix())
	runN(N, name, IMAGE, CMD)
}
func tb() {
	name := fmt.Sprintf("tb-%d", time.Now().Unix())
	runB(B, name, IMAGE, CMD)
}
func tnb() {
	name := fmt.Sprintf("tnb-%d", time.Now().Unix())
	runBonN(B, N, name, IMAGE, CMD)
}
