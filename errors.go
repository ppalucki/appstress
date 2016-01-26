package main

import (
	"fmt"
	"log"
	"runtime"
)

func where(err error) string {
	pc, file, line, ok := runtime.Caller(2)
	fn := runtime.FuncForPC(pc)
	var name string
	if fn != nil {
		name = fn.Name()
	} else {
		name = file
	}
	if ok {
		return fmt.Sprintf("[%s:%d] %s\n", name, line, err)
	}
	return err.Error()
}

func ok(err error) {
	if err != nil {
		msg := where(err)
		panic(msg)
	}
}

func warn(err error) bool {
	if err != nil {
		msg := where(err)
		log.Printf("WARN: " + msg)
		return true
	}
	return false
}

func warnStore(err error) bool {
	if err != nil {
		msg := where(err)
		storeLog("WARN:", msg)
		return true
	}
	return false
}
