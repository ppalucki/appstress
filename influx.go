package main

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/influxdb/influxdb/models"
)

var (
	points chan string // communication channel for store function
)

// openFile writer which  stores (appends) data to influx.data file
func openFile() io.Writer {
	const filename = "influx.data"
	writer, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0666)
	ok(err)
	return writer
}

// openInflux goroutine and returns writer
func openInflux() io.Writer {
	pr, pw := io.Pipe()
	scanner := bufio.NewScanner(pr)
	client := http.Client{}
	go func() {
		for scanner.Scan() {
			wg.Add(1)
			body := bytes.NewBufferString(scanner.Text())
			req, err := http.NewRequest("POST", "http://localhost:8086/write?db=docker", body)
			ok(err)

			resp, err := client.Do(req)
			warn(err)
			if err != nil && resp != nil {
				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Printf("influx resposone warn: %s\n", b)
				}
			}
			wg.Done()
		}

	}()
	return pw
}

// start goroutines that move points to file and optionally to influxdb
func init() {
	wg.Add(1)
	points = make(chan string)
	writer := io.MultiWriter(openInflux(), openFile())
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

// store point in points channel serialized as line protocol from influx
func store(name string, tags map[string]string, fields map[string]interface{}) {
	if tags == nil {
		tags = map[string]string{"name": NAME}
	} else {
		tags["name"] = NAME
	}

	p, err := models.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		panic(err)
	}

	points <- p.String() + "\n"
}
