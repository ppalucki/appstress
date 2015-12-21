package influx

import (
	"io/ioutil"
	"testing"
	"time"
)

func Test(t *testing.T) {
	err := New()
	if err != nil {
		t.Fatal(err)
	}

	err = Store("x", nil, map[string]interface{}{"foo": 2}, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	f, err := ioutil.TempFile("", "influx")
	if err != nil {
		t.Fatal(err)
	}

	err = Save(f)
	if err != nil {
		t.Fatal(err)
	}

	f.Close()
	println(f.Name())

	err = Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	Show()

	err = SaveInflux("http://127.0.0.1:8086", "test")
	if err != nil {
		t.Fatal(err)
	}

}
