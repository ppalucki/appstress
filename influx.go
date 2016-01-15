package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/influxdb/influxdb/models"
)

var (
	points chan string // communication channel for store function
)

// openFile writer which  stores (appends) data to influx.data file
func openFile(filename string) io.Writer {
	log.Println("influx out file:", filename)
	writer, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	ok(err)
	return writer
}

// openInflux goroutine and returns writer
func openInflux(url string) io.Writer {
	pr, pw := io.Pipe()
	scanner := bufio.NewScanner(pr)
	//scanner.Split(MultiScanLines) // TODO: fix me
	client := http.Client{}
	go func() {
		for scanner.Scan() {
			wg.Add(1)
			body := bytes.NewBufferString(scanner.Text())
			// log.Print("INFLUX: ", body)
			req, err := http.NewRequest("POST", url, body)
			ok(err)

			resp, err := client.Do(req)
			if warn(err) && resp != nil {
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
func initInflux(influxUrl string) {
	wg.Add(1)
	points = make(chan string)

	var writer io.Writer

	u, err := url.Parse(influxUrl)
	ok(err)
	if u.Scheme == "http" {
		writer = openInflux(influxUrl)
	} else {
		writer = openFile(u.Host)
	}
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
		tags = map[string]string{"name": *NAME}
	} else {
		tags["name"] = *NAME
	}

	p, err := models.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		panic(err)
	}

	points <- p.String() + "\n"
}

func storeLog(values ...string) {
	msg := strings.Join(values, " ")
	log.Println(msg)
	store("logs", nil, map[string]interface{}{"message": msg})
}

// influxFlush copies reads data from reader into influx given by
// uses store function (line by line)
func feedInflux(srcFilename, dstUrl string) {

	if strings.Contains(dstUrl, srcFilename) {
		fmt.Printf("please specify other destination through influxUrl=%q than src file=%q\n", dstUrl, srcFilename)
		os.Exit(1)
	}

	src, err := os.Open(srcFilename)
	ok(err)

	dst := openInflux(dstUrl)
	n, err := io.Copy(dst, src)
	ok(err)
	fmt.Printf("copied %d bytes from %q to %q\n", n, srcFilename, dstUrl)
}
