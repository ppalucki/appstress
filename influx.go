package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/influxdb/influxdb/models"
)

var (
	writer io.Writer
	points chan string
)

const filename = "influx.data"

func openFile() io.Writer {
	writer, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0666)
	ok(err)
	return writer
}

func openInflux() io.Writer {
	// conn, err := net.Dial("tcp", "127.0.0.1:8083")
	// ok(err)
	r, w := io.Pipe()
	var r2 io.Reader
	r2 = io.TeeReader(r, os.Stdout)
	wg.Add(1)
	go func() {
		req, err := http.NewRequest("POST", "http://localhost:8086/write?db=docker", r2)
		ok(err)
		client := http.Client{}
		println("do....")
		resp, err := client.Do(req)
		println("after do")
		fmt.Printf("resp = %+v\n", resp)
		ok(err)
		wg.Done()
	}()
	return w
}

func init() {
	wg.Add(1)
	points = make(chan string)
	writer = io.MultiWriter(openInflux(), openFile())
	go func() {
		for {
			select {
			case s := <-points:
				_, err := io.WriteString(writer, s)
				ok(err)

			case <-quit:
				wg.Done()
				return
			}
		}
	}()

}

func store(name string, tags map[string]string, fields map[string]interface{}) {
	p, err := models.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		panic(err)
	}
	points <- p.String() + "\n"
}
