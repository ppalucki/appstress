/*
helper to access influxdb with caching (file/memory) layer
usage:
New()
Store(measurment, tags, fields, time)
Store(measurment, tags, fields, time)
SaveFile("/tmp/data.finlux")

curl -i -XPOST 'http://localhost:8086/write?db=test2' -d @/tmp/influx924390070
or
SaveInflux("http://127.0.0.1:8086", "test")

TODO: replace mutex with channels :P
*/
package influx

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	client "github.com/influxdb/influxdb/client/v2"
	models "github.com/influxdb/influxdb/models"

	"sync"
)

var (
	bp client.BatchPoints
	m  sync.RWMutex
	ch chan client.Point
)

func New() error {
	c := client.BatchPointsConfig{}
	var err error
	_bp, err := client.NewBatchPoints(c)
	if err != nil {
		return err
	}
	m.Lock()
	defer m.Unlock()
	bp = _bp
	return nil
}

func Open(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	d, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	points, err := models.ParsePoints(d)
	if err != nil {
		return err
	}

	err = New()
	if err != nil {
		return err
	}

	defer m.Unlock()
	for _, p := range points {
		err := Store(p.Name(), p.Tags(), p.Fields(), p.Time())
		if err != nil {
			return err
		}
	}

	return nil
}

func store(name string, tags map[string]string, fields map[string]interface{}, t time.Time) error {
	p, err := client.NewPoint(name, tags, fields, t)
	if err != nil {
		return err
	}
	m.Lock()
	defer m.Unlock()
	bp.AddPoint(p)
	return nil
}

func Store(name string, tags map[string]string, fields map[string]interface{}, t time.Time) error {
	store(name, tags, fields, t)
	return nil
}

func SaveFile(fn string) error {
	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	err = Save(f)
	if err != nil {
		return err
	}
	return nil
}

func Show() error {
	return Save(os.Stdout)
}

// flush stored data to file
func Save(w io.Writer) error {
	m.RLock()
	defer m.RUnlock()
	for _, p := range bp.Points() {
		_, err := w.Write([]byte(p.String() + "\n"))
		if err != nil {
			return err
		}
	}
	return nil
}

// flus store date to influxdb
func SaveInflux(url string, db string) error {
	conf := client.HTTPConfig{
		Addr: url,
	}
	c, err := client.NewHTTPClient(conf)
	if err != nil {
		return err
	}

	q := fmt.Sprintf(`create database "%s"`, db)
	_, err = c.Query(client.Query{Command: q})
	if err != nil {
		return err
	}

	bp.SetDatabase(db)
	return c.Write(bp)
}
